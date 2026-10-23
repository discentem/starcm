package macpkg

import (
	"context"
	"path"
	"path/filepath"
	"runtime"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/functions/download"
	shelllib "github.com/discentem/starcm/libraries/shell"
)

// remoteAction is install_remote_mac_pkg(): fetch a .pkg into the local
// store with the download Fetcher, as load_remote() does for a .star file,
// then install it from there with install_mac_pkg()'s checks.
type remoteAction struct {
	pkg     *pkgAction
	fetcher *download.Fetcher
}

var _ base.Action = (*remoteAction)(nil)

func (a *remoteAction) Execute(inst *base.FunctionInstance) (*base.Result, error) {
	p, err := a.pkg.parseArgs(inst, "install_remote_mac_pkg")
	if err != nil {
		return nil, err
	}
	rawURL, _ := base.Get[string](inst, "url")
	sig, _ := base.Get[string](inst, "signature")
	endpoint, _ := base.Get[string](inst, "endpoint")
	region, _ := base.Get[string](inst, "region")
	remote, err := download.ParseRemote(rawURL, sig, endpoint, region)
	if err != nil {
		return nil, p.errorf("%v", err)
	}
	if remote.Folder || path.Ext(remote.Name) != ".pkg" {
		return nil, p.errorf("%s doesn't name a .pkg file", rawURL)
	}
	// The Fetcher checks this too, but only after the platform check.
	if (p.sha256 == "") == (sig == "") {
		return nil, p.errorf("pass exactly one of sha256 or signature")
	}
	if err := a.pkg.checkPlatform(inst, p); err != nil {
		return nil, err
	}
	p.name = rawURL
	// An installed version needs nothing fetched.
	if r, err := a.pkg.alreadyInstalled(p); r != nil || err != nil {
		return r, err
	}

	// Always into the store: no SaveTo, so the package installed is the one
	// stored under its verified sha256, which install re-hashes.
	got, err := a.fetcher.FetchRemote(inst.Ctx, remote, download.Request{
		Function: "install_remote_mac_pkg",
		Label:    p.label,
		FileName: remote.Name,
		SHA256:   p.sha256,
	})
	if err != nil {
		return nil, err
	}
	if p.path, err = filepath.Abs(got.Path); err != nil {
		return nil, err
	}
	return a.pkg.install(p)
}

// remoteArgs are install_remote_mac_pkg()'s own arguments.
var remoteArgs = []base.ArgPair{
	base.Arg[string]("url"),
	base.Arg[string]("signer"),
	base.OptionalArg[string]("sha256"),
	base.OptionalArg[string]("signature"),
	base.OptionalArgAllowNone[bool]("require_notarized"),
	base.OptionalArg[string]("target"),
	base.OptionalArg[string]("id"),
	base.OptionalArg[string]("version"),
	base.OptionalArg[string]("endpoint"),
	base.OptionalArg[string]("region"),
}

// NewRemote returns the install_remote_mac_pkg() builtin, which fetches
// with f and runs pkgutil and installer with executor. opts are
// install_mac_pkg()'s.
//
//	install_remote_mac_pkg(
//	    "https://releases.example.com/agent-1.2.0.pkg",
//	    signature = "https://releases.example.com/agent-1.2.0.pkg.sig",
//	    signer    = "Developer ID Installer: Example Corp (ABCDE12345)",
//	    label     = "install agent",
//	)
func NewRemote(ctx context.Context, f *download.Fetcher, executor shelllib.Executor, opts ...Option) *base.Module {
	a := &pkgAction{executor: executor, goos: runtime.GOOS}
	for _, opt := range opts {
		opt(a)
	}
	return base.NewModule(
		ctx,
		"install_remote_mac_pkg",
		remoteArgs,
		&remoteAction{pkg: a, fetcher: f},
	)
}
