package sign

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/libraries/signing"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

type signAction struct {
	fsys     afero.Fs
	deployed bool
}

var _ base.Runnable = (*signAction)(nil)

// Run writes path + ".sig", like `starcm sign`. ed25519 signatures are
// deterministic, so re-signing unchanged content with the same key and expiry
// reports changed = False. An expiry given as a duration ("30d") is counted
// from now, so it changes the signature on every run.
func (a *signAction) Run(ctx context.Context, workingDirectory string, moduleName string, thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	if a.deployed {
		return nil, errors.New("sign() is disabled when starcm runs with --trusted-keys: signing belongs on the machine that holds the key, not where configuration is deployed")
	}
	var p string
	if args.Len() == 1 {
		p = args.Index(0).(starlark.String).GoString()
	} else {
		v, err := starlarkhelpers.FindValueinKwargs(kwargs, "path")
		if err != nil {
			return nil, err
		}
		p = *v
	}
	keyPath, err := starlarkhelpers.FindValueinKwargs(kwargs, "key")
	if err != nil {
		return nil, err
	}
	keyID, err := starlarkhelpers.FindValueInKwargsWithDefault(kwargs, "key_id", "")
	if err != nil {
		return nil, err
	}
	expiresArg, err := starlarkhelpers.FindValueinKwargs(kwargs, "expires")
	if err != nil {
		return nil, err
	}
	expires, err := signing.ParseExpiry(*expiresArg, time.Now())
	if err != nil {
		return nil, err
	}
	sig, changed, err := signing.SignFile(a.fsys, *keyPath, *keyID, p, expires)
	if err != nil {
		return nil, err
	}
	sigPath := p + signing.SignatureSuffix
	msg := fmt.Sprintf("signed %s with key %q until %s -> %s", p, sig.KeyID, sig.Expires.Format(time.RFC3339), sigPath)
	return &base.Result{Label: moduleName, Success: true, Changed: changed, Message: &msg, Return: starlark.String(sig.KeyID)}, nil
}

// New returns the sign() builtin. deployed is true when starcm runs with
// --trusted-keys, which disables it.
//
//	sign("main.star", key = vars["key"], expires = "30d", label = "sign main.star")
func New(ctx context.Context, fsys afero.Fs, deployed bool) *base.Module {
	return base.NewModule(
		ctx,
		"sign",
		[]base.ArgPair{
			base.Arg[string]("path"),
			base.Arg[string]("key"),
			base.Arg[string]("expires"),
			base.Arg[string]("key_id?"),
		},
		&signAction{fsys: fsys, deployed: deployed},
	)
}
