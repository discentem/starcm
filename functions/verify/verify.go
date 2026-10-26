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
	"go.starlark.net/starlarkstruct"
)

// check verifies path against its signature file using keyring, and
// returns a verification struct. A file that doesn't verify is an answer,
// not an error: verified is False and error says why, so scripts decide
// whether to fail().
func check(fsys afero.Fs, keyring *signing.Keyring, inst *base.FunctionInstance) (starlark.Value, error) {
	if keyring == nil {
		return nil, errors.New("verify() needs trusted keys: run starcm with --trusted-keys")
	}
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
	failed := func(err error) starlark.Value {
		return verification(false, starlark.None, starlark.None, starlark.None, starlark.String(fmt.Sprintf("%s: %s", path, err)))
	}
	data, err := afero.ReadFile(fsys, path)
	if err != nil {
		return failed(err), nil
	}
	sigFile, err := afero.ReadFile(fsys, sigPath)
	if err != nil {
		return failed(fmt.Errorf("reading signature: %w", err)), nil
	}
	sig, err := keyring.Verify(data, sigFile)
	if err != nil {
		return failed(err), nil
	}
	return verification(true,
		starlark.String(sig.KeyID),
		starlark.String(sig.Release),
		starlark.String(sig.Expires.Format(time.RFC3339)),
		starlark.None,
	), nil
}

func verification(verified bool, keyID, release, expires, errMsg starlark.Value) starlark.Value {
	return starlarkstruct.FromStringDict(starlark.String("verification"), starlark.StringDict{
		"verified": starlark.Bool(verified),
		"key_id":   keyID,
		"release":  release,
		"expires":  expires,
		"error":    errMsg,
	})
}

// New returns verify(), a query that checks a file against its signature
// with the --trusted-keys keyring. keyring may be nil, in which case
// verify() stops the run.
//
//	v = verify("/opt/agent.msi")  # reads /opt/agent.msi.sig
//	v.verified, v.key_id, v.error
func New(ctx context.Context, fsys afero.Fs, keyring *signing.Keyring) *base.Query {
	return base.NewQuery(
		ctx,
		"verify",
		[]base.ArgPair{
			base.Arg[string]("path"),
			base.OptionalArg[string]("signature"),
		},
		func(inst *base.FunctionInstance) (starlark.Value, error) {
			return check(fsys, keyring, inst)
		},
	)
}
