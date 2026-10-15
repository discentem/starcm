package verify

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

type verifyAction struct {
	fsys    afero.Fs
	keyring *signing.Keyring
}

var _ base.Runnable = (*verifyAction)(nil)

// Run checks path against its signature file using the --trusted-keys
// keyring. A file that doesn't verify is a failed result rather than an
// error, like exec(); scripts decide whether to fail().
func (a *verifyAction) Run(ctx context.Context, workingDirectory string, moduleName string, thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	if a.keyring == nil {
		return nil, errors.New("verify() needs trusted keys: run starcm with --trusted-keys")
	}
	var path string
	if args.Len() == 1 {
		path = args.Index(0).(starlark.String).GoString()
	} else {
		p, err := starlarkhelpers.FindValueinKwargs(kwargs, "path")
		if err != nil {
			return nil, err
		}
		path = *p
	}
	sigPath, err := starlarkhelpers.FindValueInKwargsWithDefault(kwargs, "signature", path+signing.SignatureSuffix)
	if err != nil {
		return nil, err
	}

	failed := func(err error) *base.Result {
		return &base.Result{Label: moduleName, Success: false, Error: fmt.Errorf("%s: %w", path, err)}
	}
	data, err := afero.ReadFile(a.fsys, path)
	if err != nil {
		return failed(err), nil
	}
	sigFile, err := afero.ReadFile(a.fsys, *sigPath)
	if err != nil {
		return failed(fmt.Errorf("reading signature: %w", err)), nil
	}
	sig, err := a.keyring.Verify(data, sigFile)
	if err != nil {
		return failed(err), nil
	}

	msg := fmt.Sprintf("%s verified with key %q, signature expires %s", path, sig.KeyID, sig.Expires.Format(time.RFC3339))
	return &base.Result{
		Label:   moduleName,
		Success: true,
		Message: &msg,
		Return:  starlark.String(sig.KeyID),
	}, nil
}

// New returns the verify() builtin. keyring comes from --trusted-keys and
// may be nil, in which case verify() errors.
//
//	r = verify("agent.msi", label = "check agent")  # reads agent.msi.sig
//	r.success, r.value (the key id), r.error
func New(ctx context.Context, fsys afero.Fs, keyring *signing.Keyring) *base.Module {
	return base.NewModule(
		ctx,
		"verify",
		[]base.ArgPair{
			base.Arg[string]("path"),
			base.Arg[string]("signature?"),
		},
		&verifyAction{fsys: fsys, keyring: keyring},
	)
}
