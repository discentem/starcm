package loading

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/functions/download"
	"github.com/discentem/starcm/libraries/signing"
)

// RemoteAction is load_remote(): fetch into the local store with the
// download Fetcher, then load_dynamic() of the stored file. With
// folder = True, url (ending in "/") is a folder: FetchFolder assembles it in
// the store and module, a path in its manifest, is loaded from there. Pinning, signatures and the store
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
	sha, _ := base.Get[string](inst, "sha256")
	sig, _ := base.Get[string](inst, "signature")
	endpoint, _ := base.Get[string](inst, "endpoint")
	region, _ := base.Get[string](inst, "region")
	remote, err := download.ParseRemote(rawURL, sig, endpoint, region)
	if err != nil {
		return nil, fail("%v", err)
	}
	folder, _ := base.Get[bool](inst, "folder")
	module, moduleGiven := base.Get[string](inst, "module")
	fileName := remote.Name
	switch {
	case folder:
		if !moduleGiven {
			module = "main.star"
		}
		if err := signing.CheckManifestPath(module); err != nil {
			return nil, fail("module: %v", err)
		}
		if path.Ext(module) != ".star" {
			return nil, fail("module %q doesn't name a .star file", module)
		}
		fileName = ""
	case moduleGiven:
		return nil, fail("module is only for folder = True")
	case remote.Folder:
		// The Fetcher says to pass folder = True.
	case path.Ext(remote.Name) != ".star":
		return nil, fail("%s doesn't name a .star file", rawURL)
	}

	// Always into the store: no SaveTo, so the loaded file is the one
	// stored under its verified sha256.
	got, err := a.Fetcher.FetchRemote(inst.Ctx, remote, download.Request{Function: "load_remote", Label: label, FileName: fileName, SHA256: sha, Folder: folder})
	if err != nil {
		return nil, err
	}
	stored := got.Path
	if folder {
		if _, listed := got.Manifest.Files[module]; !listed {
			return nil, fail("module %q is not in %s%s", module, rawURL, signing.ManifestName)
		}
		stored = filepath.Join(got.Path, filepath.FromSlash(module))
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
		msg = fmt.Sprintf("loaded %s%s from %s", rawURL, module, stored)
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
//	r = load_remote("https://config.example.com/agent/", folder = True, module = "main.star", signature = "https://config.example.com/agent/starcm-manifest.json.sig", label = "agent")
func NewRemote(ctx context.Context, f *download.Fetcher) *base.Module {
	return base.NewModule(
		ctx,
		"load_remote",
		[]base.ArgPair{
			base.Arg[string]("url"),
			base.OptionalArgAllowNone[bool]("folder"),
			base.OptionalArg[string]("module"),
			base.OptionalArg[string]("sha256"),
			base.OptionalArg[string]("signature"),
			base.OptionalArg[string]("endpoint"),
			base.OptionalArg[string]("region"),
		},
		&RemoteAction{Fetcher: f},
	)
}
