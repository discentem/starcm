package sign

import (
	"context"
	"errors"
	"time"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/libraries/signing"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

type signAction struct {
	fsys     afero.Fs
	deployed bool
}

var _ base.Action = (*signAction)(nil)

// Execute writes path + ".sig", like `starcm sign`. ed25519 signatures are
// deterministic, so re-signing unchanged content with the same key and expiry
// reports changed = False. An expiry given as a duration ("30d") is counted
// from now, so it changes the signature on every run.
func (a *signAction) Execute(inst *base.FunctionInstance) (*base.Result, error) {
	if a.deployed {
		return nil, errors.New("sign() is disabled when starcm runs with --trusted-keys: signing belongs on the machine that holds the key, not where configuration is deployed")
	}
	p, _, err := inst.AbsPath("path")
	if err != nil {
		return nil, err
	}
	keyPath, _, err := inst.AbsPath("key")
	if err != nil {
		return nil, err
	}
	keyID, _ := base.Get[string](inst, "key_id")
	release, _ := base.Get[string](inst, "release")
	expiresArg, _ := base.Get[string](inst, "expires")
	expires, err := signing.ParseExpiry(expiresArg, time.Now())
	if err != nil {
		return nil, err
	}
	sig, changed, err := signing.SignFile(a.fsys, keyPath, keyID, release, p, expires)
	if err != nil {
		return nil, err
	}
	msg := signing.Describe(p, sig)
	return &base.Result{Label: inst.Label, Success: true, Changed: changed, Message: &msg, Return: starlark.String(sig.KeyID)}, nil
}

// New returns the sign() builtin. deployed is true when starcm runs with
// --trusted-keys, which disables it.
//
//	sign(here("main.star"), key = vars["key"], expires = "30d", label = "sign main.star")
func New(ctx context.Context, fsys afero.Fs, deployed bool) *base.Module {
	return base.NewModule(
		ctx,
		"sign",
		[]base.ArgPair{
			base.Arg[string]("path"),
			base.Arg[string]("key"),
			base.Arg[string]("expires"),
			base.OptionalArg[string]("key_id"),
			base.OptionalArg[string]("release"),
		},
		&signAction{fsys: fsys, deployed: deployed},
	)
}
