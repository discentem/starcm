package verify

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/libraries/signing"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

type verifyAction struct {
	fsys    afero.Fs
	keyring *signing.Keyring
}

var _ base.Action = (*verifyAction)(nil)

// Execute checks path against its signature file using the --trusted-keys
// keyring. A file that doesn't verify is a failed result rather than an
// error, like exec(); scripts decide whether to fail().
func (a *verifyAction) Execute(inst *base.FunctionInstance) (*base.Result, error) {
	if a.keyring == nil {
		return nil, errors.New("verify() needs trusted keys: run starcm with --trusted-keys")
	}
	moduleName := inst.Label
	path, _, err := inst.AbsPath("path")
	if err != nil {
		return nil, err
	}
	sigPath, given, err := inst.AbsPath("signature")
	if err != nil {
		return nil, err
	}
	if !given {
		sigPath = path + signing.SignatureSuffix
	}

	failed := func(err error) *base.Result {
		return &base.Result{Label: moduleName, Success: false, Error: fmt.Errorf("%s: %w", path, err)}
	}
	data, err := afero.ReadFile(a.fsys, path)
	if err != nil {
		return failed(err), nil
	}
	sigFile, err := afero.ReadFile(a.fsys, sigPath)
	if err != nil {
		return failed(fmt.Errorf("reading signature: %w", err)), nil
	}
	sig, err := a.keyring.Verify(data, sigFile)
	if err != nil {
		return failed(err), nil
	}

	var release string
	if sig.Release != "" {
		release = fmt.Sprintf(" for release %q", sig.Release)
	}
	msg := fmt.Sprintf("%s verified with key %q%s, signature expires %s", path, sig.KeyID, release, sig.Expires.Format(time.RFC3339))
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
//	r = verify("/opt/agent.msi", label = "check agent")  # reads /opt/agent.msi.sig
//	r.success, r.value (the key id), r.error
func New(ctx context.Context, fsys afero.Fs, keyring *signing.Keyring) *base.Module {
	return base.NewModule(
		ctx,
		"verify",
		[]base.ArgPair{
			base.Arg[string]("path"),
			base.OptionalArg[string]("signature"),
		},
		&verifyAction{fsys: fsys, keyring: keyring},
	)
}
