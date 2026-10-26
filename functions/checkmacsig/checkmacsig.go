// Package checkmacsig implements check_mac_signature(), which reports who
// signed a macOS package, binary or app, and stops the run unless it's the
// developer the configuration names.
package checkmacsig

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/libraries/fileutils"
	"github.com/discentem/starcm/libraries/logging"
	"github.com/discentem/starcm/libraries/macsig"
	shelllib "github.com/discentem/starcm/libraries/shell"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

type checkAction struct {
	executor shelllib.Executor
	goos     string
	// checkStored verifies path if it names a file in the local store.
	// Nil means no store.
	checkStored func(path string) error
}

func (a *checkAction) check(inst *base.FunctionInstance) (starlark.Value, error) {
	rawPath, _ := base.Get[string](inst, "path")
	signer, _ := base.Get[string](inst, "signer")
	if strings.TrimSpace(signer) == "" {
		return nil, inst.Errorf("signer must name the certificate, e.g. \"Developer ID Application: Example Corp (ABCDE12345)\"")
	}
	requireNotarized := true
	if v, ok := base.Get[bool](inst, "require_notarized"); ok {
		requireNotarized = v
	}
	kindArg, given := base.Get[string](inst, "kind")
	if !given {
		kindArg = macsig.KindAuto
	}
	kind, err := macsig.ResolveKind(rawPath, kindArg)
	if err != nil {
		return nil, inst.Errorf("%w", err)
	}
	if a.goos != "darwin" {
		return nil, inst.Errorf("check_mac_signature() only works on macOS")
	}
	// Whether a path is absolute depends on the OS, so check it once we
	// know we're on macOS.
	path, _, err := inst.AbsPath("path")
	if err != nil {
		return nil, err
	}
	if a.checkStored != nil {
		if err := a.checkStored(path); err != nil {
			return nil, inst.Errorf("%w", err)
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, inst.Errorf("%w", err)
	}
	// Check a private copy of a file, so the sha256 reported is of the bytes
	// whose signature was checked. A bundle is checked where it is.
	checked, sum := path, ""
	if !info.IsDir() {
		dir, err := os.MkdirTemp("", "starcm-mac-sig-")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(dir)
		checked = filepath.Join(dir, filepath.Base(path))
		if sum, err = fileutils.CopyNew(path, checked, 0o600); err != nil {
			return nil, inst.Errorf("%w", err)
		}
	}

	sig, err := macsig.New(a.executor).Read(checked, kind)
	if err != nil {
		return nil, inst.Errorf("%s: %w", path, err)
	}
	if err := sig.Check(signer, requireNotarized); err != nil {
		return nil, inst.Errorf("%s: %w", path, err)
	}
	fields := sig.Fields()
	fields["sha256"] = starlark.String(sum)
	fields["kind"] = starlark.String(kind)
	logging.Log("check_mac_signature", nil, "info", "%s is signed by %q", path, sig.Chain[0].Name)
	return starlarkstruct.FromStringDict(starlark.String("mac_signature"), fields), nil
}

// checkArgs are check_mac_signature()'s own arguments.
var checkArgs = []base.ArgPair{
	base.Arg[string]("path"),
	base.Arg[string]("signer"),
	base.OptionalArgAllowNone[bool]("require_notarized"),
	base.OptionalArg[string]("kind"),
}

// Option configures check_mac_signature().
type Option func(*checkAction)

// WithStoreCheck makes check_mac_signature() call check on the path first,
// so a file from the local store is re-hashed before it's checked.
func WithStoreCheck(check func(path string) error) Option {
	return func(a *checkAction) { a.checkStored = check }
}

// WithGOOS replaces the operating system check_mac_signature() thinks it
// runs on, for tests.
func WithGOOS(goos string) Option {
	return func(a *checkAction) { a.goos = goos }
}

// New returns check_mac_signature(), a query that runs pkgutil or codesign
// with executor and returns the signature, or stops the run unless the
// signer is the one named.
//
//	check_mac_signature(terraform_path, signer = "Developer ID Application: Hashicorp, Inc. (D38WU7D763)")
func New(ctx context.Context, executor shelllib.Executor, opts ...Option) *base.Query {
	a := &checkAction{executor: executor, goos: runtime.GOOS}
	for _, opt := range opts {
		opt(a)
	}
	return base.NewQuery(ctx, "check_mac_signature", checkArgs, a.check)
}
