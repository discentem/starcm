package loading

import (
	"context"
	"fmt"
	neturl "net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/functions/download"
	"github.com/discentem/starcm/libraries/signing"
)

// RemoteAction is load_remote(): fetch into the local store with the
// download Fetcher, then load_dynamic() of the stored file. A url ending in
// "/" is a folder: FetchFolder assembles it in the store and entry, a path
// in its manifest, is loaded from there. Pinning, signatures and the store
// are the Fetcher's; loading, --trusted-keys and the once-per-run cache are
// the loader's.
type RemoteAction struct {
	Fetcher *download.Fetcher
}

var _ base.Action = (*RemoteAction)(nil)

func (a *RemoteAction) Execute(inst *base.FunctionInstance) (*base.Result, error) {
	thread, label := inst.Thread, inst.Label
	if thread == nil || thread.Load == nil {
		return nil, fmt.Errorf("%s: thread.Load is nil", label)
	}
	fail := func(format string, v ...any) error {
		return fmt.Errorf("load_remote(label=%q): %s", label, fmt.Sprintf(format, v...))
	}

	rawURL, _ := base.Get[string](inst, "url")
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return nil, fail("%v", err)
	}
	folder := download.IsFolder(u.Path)
	entry, entryGiven := base.Get[string](inst, "entry")
	name := path.Base(u.Path)
	switch {
	case folder:
		if !entryGiven {
			entry = "main.star"
		}
		if err := signing.CheckManifestPath(entry); err != nil {
			return nil, fail("entry: %v", err)
		}
		if path.Ext(entry) != ".star" {
			return nil, fail("entry %q doesn't name a .star file", entry)
		}
		name = ""
	case entryGiven:
		return nil, fail("entry is only for a folder: a url ending in /")
	case path.Ext(name) != ".star":
		return nil, fail("%s doesn't name a .star file", rawURL)
	}
	sha, _ := base.Get[string](inst, "sha256")
	sig, _ := base.Get[string](inst, "signature")
	endpoint, _ := base.Get[string](inst, "endpoint")
	region, _ := base.Get[string](inst, "region")

	// Always into the store: no SaveTo, so the loaded file is the one
	// stored under its verified sha256.
	req := download.Request{Function: "load_remote", Label: label, FileName: name, SHA256: sha, Signature: sig}
	var got download.Fetched
	switch u.Scheme {
	case "http", "https":
		if endpoint != "" || region != "" {
			return nil, fail("endpoint and region are only for s3:// URLs")
		}
		got, err = a.Fetcher.FetchURL(inst.Ctx, rawURL, req)
	case "s3":
		key := strings.TrimPrefix(u.Path, "/")
		if u.Host == "" || key == "" {
			return nil, fail("want s3://bucket/key, got %s", rawURL)
		}
		if sig != "" {
			// FetchS3 takes the signature's key; it must be in the same
			// bucket as the module.
			su, err := neturl.Parse(sig)
			if err != nil || su.Scheme != "s3" || su.Host != u.Host || strings.TrimPrefix(su.Path, "/") == "" {
				return nil, fail("signature %s must be an s3:// URL in bucket %q", sig, u.Host)
			}
			req.Signature = strings.TrimPrefix(su.Path, "/")
		}
		got, err = a.Fetcher.FetchS3(inst.Ctx, download.S3Object{Bucket: u.Host, Key: key, Endpoint: endpoint, Region: region}, req)
	default:
		return nil, fail("url must start with http://, https:// or s3://, got %s", rawURL)
	}
	if err != nil {
		return nil, err
	}
	stored := got.Path
	if folder {
		if _, listed := got.Manifest.Files[entry]; !listed {
			return nil, fail("entry %q is not in %s%s", entry, rawURL, signing.ManifestName)
		}
		stored = filepath.Join(got.Path, filepath.FromSlash(entry))
	}
	// An absolute path, so the loader doesn't resolve it against the
	// calling file's directory.
	modulePath, err := filepath.Abs(stored)
	if err != nil {
		return nil, err
	}
	d, err := loadGlobals(thread, filepath.ToSlash(modulePath))
	if err != nil {
		return nil, err
	}
	msg := fmt.Sprintf("loaded %s from %s", rawURL, stored)
	if folder {
		msg = fmt.Sprintf("loaded %s%s from %s", rawURL, entry, stored)
	}
	return &base.Result{
		Label:   label,
		Success: true,
		Changed: got.Changed,
		Message: &msg,
		Return:  d,
	}, nil
}

// NewRemote returns the load_remote() builtin, which fetches with f.
//
//	r = load_remote("https://config.example.com/agent.star", signature = "https://config.example.com/agent.star.sig", label = "agent")
//	r.value["configure"]()
//	r = load_remote("https://config.example.com/agent/", entry = "main.star", signature = "https://config.example.com/agent/starcm-manifest.json.sig", label = "agent")
func NewRemote(ctx context.Context, f *download.Fetcher) *base.Module {
	return base.NewModule(
		ctx,
		"load_remote",
		[]base.ArgPair{
			base.Arg[string]("url"),
			base.OptionalArg[string]("entry"),
			base.OptionalArg[string]("sha256"),
			base.OptionalArg[string]("signature"),
			base.OptionalArg[string]("endpoint"),
			base.OptionalArg[string]("region"),
		},
		&RemoteAction{Fetcher: f},
	)
}
