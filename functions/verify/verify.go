package verify

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/libraries/paths"
	"github.com/discentem/starcm/libraries/signing"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

var errNoKeys = errors.New("needs trusted keys: run starcm with --trusted-keys")

// file checks path against the signature file sigPath with keyring. The
// error says why the file didn't verify.
//
// There is no single-file verify(): checking a file and then using it in a
// separate step leaves a gap in which it can be swapped. download,
// load_remote and the install functions verify the bytes they use;
// verify_all is for auditing files that arrived some other way.
func file(fsys afero.Fs, keyring *signing.Keyring, path, sigPath string) (starlark.Value, error) {
	data, err := afero.ReadFile(fsys, path)
	if err != nil {
		return nil, err
	}
	sigFile, err := afero.ReadFile(fsys, sigPath)
	if err != nil {
		return nil, fmt.Errorf("reading signature: %w", err)
	}
	sig, err := keyring.Verify(data, sigFile)
	if err != nil {
		return nil, err
	}
	return starlarkstruct.FromStringDict(starlark.String("verification"), starlark.StringDict{
		"path":    starlark.String(path),
		"key_id":  starlark.String(sig.KeyID),
		"release": starlark.String(sig.Release),
		"expires": starlark.String(sig.Expires.Format(time.RFC3339)),
	}), nil
}

// NewAll returns verify_all(), a query that checks every file in paths
// against the signature next to it (path + ".sig"). It checks them all
// before deciding: if any don't verify, it stops the run once, listing each
// one and why. Otherwise it returns their verifications, in order.
//
//	verify_all(path.glob(here("payloads/*.pkg")))
func NewAll(ctx context.Context, fsys afero.Fs, keyring *signing.Keyring) *base.Query {
	return base.NewQuery(
		ctx,
		"verify_all",
		[]base.ArgPair{base.Arg[*starlark.List]("paths")},
		func(inst *base.FunctionInstance) (starlark.Value, error) {
			if keyring == nil {
				return nil, inst.Errorf("%w", errNoKeys)
			}
			list, _ := base.Get[*starlark.List](inst, "paths")
			var (
				out    []starlark.Value
				failed []string
			)
			for i := 0; i < list.Len(); i++ {
				s, ok := starlark.AsString(list.Index(i))
				if !ok {
					return nil, inst.Errorf("paths[%d]: got %s, want string", i, list.Index(i).Type())
				}
				path, err := paths.Abs(s)
				if err != nil {
					return nil, inst.Errorf("paths[%d]: %w", i, err)
				}
				v, err := file(fsys, keyring, path, path+signing.SignatureSuffix)
				if err != nil {
					failed = append(failed, fmt.Sprintf("  %s: %s", path, err))
					continue
				}
				out = append(out, v)
			}
			if len(failed) > 0 {
				return nil, inst.Errorf("%d of %d files didn't verify:\n%s", len(failed), list.Len(), strings.Join(failed, "\n"))
			}
			return starlark.NewList(out), nil
		},
	)
}
