package loading

import (
	"context"
	"fmt"
	neturl "net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/discentem/starcm/functions/base"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"go.starlark.net/starlark"
)

// RemoteAction is load_remote(): download() or s3_download() into the local
// store, then load_dynamic() of the stored file. Pinning, signatures and the
// store are download's; loading, --trusted-keys and the once-per-run cache
// are the loader's.
type RemoteAction struct {
	// HTTP is download()'s action, for http:// and https:// URLs.
	HTTP base.Runnable
	// S3 is s3_download()'s action, for s3:// URLs.
	S3 base.Runnable
}

var _ base.Runnable = (*RemoteAction)(nil)

func (a *RemoteAction) Run(ctx context.Context, workingDirectory string, label string, thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	if thread == nil || thread.Load == nil {
		return nil, fmt.Errorf("%s: thread.Load is nil", label)
	}
	fail := func(format string, v ...any) error {
		return fmt.Errorf("load_remote(label=%q): %s", label, fmt.Sprintf(format, v...))
	}

	var rawURL string
	if args.Len() == 1 {
		rawURL = args.Index(0).(starlark.String).GoString()
	} else {
		v, err := starlarkhelpers.FindValueinKwargs(kwargs, "url")
		if err != nil {
			return nil, err
		}
		rawURL = *v
	}
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return nil, fail("%v", err)
	}
	name := path.Base(u.Path)
	if path.Ext(name) != ".star" {
		return nil, fail("%s doesn't name a .star file", rawURL)
	}

	opt := func(key string) (string, error) {
		v, err := starlarkhelpers.FindValueInKwargsWithDefault(kwargs, key, "")
		if err != nil {
			return "", err
		}
		return *v, nil
	}
	pin := map[string]string{}
	for _, k := range []string{"sha256", "signature", "endpoint", "region"} {
		if pin[k], err = opt(k); err != nil {
			return nil, err
		}
	}

	// Always into the store: no save_to, so the loaded file is the one
	// stored under its verified sha256.
	fetchArgs := []starlark.Tuple{{starlark.String("file_name"), starlark.String(name)}}
	add := func(k, v string) {
		if v != "" {
			fetchArgs = append(fetchArgs, starlark.Tuple{starlark.String(k), starlark.String(v)})
		}
	}
	var fetch base.Runnable
	switch u.Scheme {
	case "http", "https":
		if pin["endpoint"] != "" || pin["region"] != "" {
			return nil, fail("endpoint and region are only for s3:// URLs")
		}
		fetch = a.HTTP
		add("url", rawURL)
		add("sha256", pin["sha256"])
		add("signature", pin["signature"])
	case "s3":
		key := strings.TrimPrefix(u.Path, "/")
		if u.Host == "" || key == "" {
			return nil, fail("want s3://bucket/key, got %s", rawURL)
		}
		fetch = a.S3
		add("bucket", u.Host)
		add("key", key)
		add("sha256", pin["sha256"])
		add("endpoint", pin["endpoint"])
		add("region", pin["region"])
		if sig := pin["signature"]; sig != "" {
			// s3_download takes the signature's key; it must be in the same
			// bucket as the module.
			su, err := neturl.Parse(sig)
			if err != nil || su.Scheme != "s3" || su.Host != u.Host || strings.TrimPrefix(su.Path, "/") == "" {
				return nil, fail("signature %s must be an s3:// URL in bucket %q", sig, u.Host)
			}
			add("signature", strings.TrimPrefix(su.Path, "/"))
		}
	default:
		return nil, fail("url must start with http://, https:// or s3://, got %s", rawURL)
	}
	if fetch == nil {
		return nil, fail("%s:// URLs aren't supported here", u.Scheme)
	}

	got, err := fetch.Run(ctx, workingDirectory, label, thread, nil, fetchArgs)
	if err != nil {
		return nil, err
	}
	stored, ok := got.Return.(starlark.String)
	if !ok {
		return nil, fail("download returned no stored path")
	}
	// An absolute path, so the loader doesn't resolve it against the
	// calling file's directory.
	modulePath, err := filepath.Abs(stored.GoString())
	if err != nil {
		return nil, err
	}
	d, err := loadGlobals(thread, filepath.ToSlash(modulePath))
	if err != nil {
		return nil, err
	}
	msg := fmt.Sprintf("loaded %s from %s", rawURL, stored.GoString())
	return &base.Result{
		Label:   label,
		Success: true,
		Changed: got.Changed,
		Message: &msg,
		Return:  d,
	}, nil
}

// NewRemote returns the load_remote() builtin. fetchHTTP and fetchS3 are
// download()'s and s3_download()'s actions.
//
//	r = load_remote("https://config.example.com/agent.star", signature = "https://config.example.com/agent.star.sig", label = "agent")
//	r.value["configure"]()
func NewRemote(ctx context.Context, fetchHTTP, fetchS3 base.Runnable) *base.Module {
	return base.NewModule(
		ctx,
		"load_remote",
		[]base.ArgPair{
			base.Arg[string]("url"),
			base.OptionalArg[string]("sha256"),
			base.OptionalArg[string]("signature"),
			base.OptionalArg[string]("endpoint"),
			base.OptionalArg[string]("region"),
		},
		&RemoteAction{HTTP: fetchHTTP, S3: fetchS3},
	)
}
