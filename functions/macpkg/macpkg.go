// Package macpkg implements install_mac_pkg(), which installs a macOS .pkg
// only after checking that it's signed, through Apple, by the developer
// certificate the configuration names.
package macpkg

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/libraries/macsig"
	starcmsha256 "github.com/discentem/starcm/libraries/sha256"
	shelllib "github.com/discentem/starcm/libraries/shell"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"go.starlark.net/starlark"
)

type pkgAction struct {
	executor shelllib.Executor
	goos     string
	// checkStored verifies path if it names a file in the local store.
	// Nil means no store.
	checkStored func(path string) error
}

var _ base.Runnable = (*pkgAction)(nil)

// run runs a command with the executor and returns its combined stdout and
// stderr. err is non-nil when the command couldn't start or exited non-zero.
func (a *pkgAction) run(name string, args ...string) ([]byte, error) {
	a.executor.Command(name, args...)
	return a.executor.CombinedOutput()
}

type parsedArgs struct {
	path             string
	signer           string
	sha256           string // "" when not pinned
	requireNotarized bool
	target           string
	id, version      string
	whatIf           bool
}

func (a *pkgAction) parseArgs(label string, args starlark.Tuple, kwargs []starlark.Tuple) (*parsedArgs, error) {
	p := &parsedArgs{}
	if args.Len() == 1 {
		p.path = args.Index(0).(starlark.String).GoString()
	} else {
		v, err := starlarkhelpers.FindValueinKwargs(kwargs, "path")
		if err != nil {
			return nil, err
		}
		p.path = *v
	}
	signer, err := starlarkhelpers.FindValueinKwargs(kwargs, "signer")
	if err != nil {
		return nil, err
	}
	if p.signer = strings.TrimSpace(*signer); p.signer == "" {
		return nil, fmt.Errorf("install_mac_pkg(label=%q): signer must name the certificate, e.g. \"Developer ID Installer: Example Corp (ABCDE12345)\"", label)
	}
	sum, err := starlarkhelpers.FindValueInKwargsWithDefault(kwargs, "sha256", "")
	if err != nil {
		return nil, err
	}
	if p.sha256 = strings.ToLower(*sum); p.sha256 != "" && !sha256Hex.MatchString(p.sha256) {
		return nil, fmt.Errorf("install_mac_pkg(label=%q): sha256 %q is not 64 hex characters", label, *sum)
	}
	if p.requireNotarized, err = starlarkhelpers.FindBoolInKwargs(kwargs, "require_notarized", true); err != nil {
		return nil, err
	}
	target, err := starlarkhelpers.FindValueInKwargsWithDefault(kwargs, "target", "/")
	if err != nil {
		return nil, err
	}
	p.target = *target
	id, err := starlarkhelpers.FindValueInKwargsWithDefault(kwargs, "id", "")
	if err != nil {
		return nil, err
	}
	version, err := starlarkhelpers.FindValueInKwargsWithDefault(kwargs, "version", "")
	if err != nil {
		return nil, err
	}
	p.id, p.version = *id, *version
	if (p.id == "") != (p.version == "") {
		return nil, fmt.Errorf("install_mac_pkg(label=%q): id and version must be given together", label)
	}
	if p.whatIf, err = starlarkhelpers.FindBoolInKwargs(kwargs, "what_if", false); err != nil {
		return nil, err
	}
	return p, nil
}

func (a *pkgAction) Run(ctx context.Context, workingDirectory string, label string, thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	p, err := a.parseArgs(label, args, kwargs)
	if err != nil {
		return nil, err
	}
	if a.goos != "darwin" {
		return nil, fmt.Errorf("install_mac_pkg(label=%q): install_mac_pkg() only works on macOS", label)
	}

	if p.id != "" {
		installed, err := a.installedVersion(p.id, p.target)
		if err != nil {
			return nil, fmt.Errorf("install_mac_pkg(label=%q): %w", label, err)
		}
		if installed == p.version {
			msg := fmt.Sprintf("%s %s is already installed", p.id, p.version)
			return &base.Result{Label: label, Success: true, Message: &msg}, nil
		}
	}

	if a.checkStored != nil {
		if err := a.checkStored(p.path); err != nil {
			return nil, fmt.Errorf("install_mac_pkg(label=%q): %w", label, err)
		}
	}

	// Check and install a private copy, so the file can't be swapped
	// between the signature check and the install.
	dir, err := os.MkdirTemp("", "starcm-mac-pkg-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	pkgPath := filepath.Join(dir, filepath.Base(p.path))
	sum, err := copyFile(p.path, pkgPath)
	if err != nil {
		return nil, fmt.Errorf("install_mac_pkg(label=%q): %w", label, err)
	}
	if p.sha256 != "" && sum != p.sha256 {
		return nil, fmt.Errorf("install_mac_pkg(label=%q): %s has sha256 %s, not the expected %s", label, p.path, sum, p.sha256)
	}

	sig, err := macsig.New(a.executor).ReadPkg(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("install_mac_pkg(label=%q): %s: %w", label, p.path, err)
	}
	if err := sig.Check(p.signer, p.requireNotarized); err != nil {
		return nil, fmt.Errorf("install_mac_pkg(label=%q): %s: %w", label, p.path, err)
	}
	value := sig.Value()

	if p.whatIf {
		msg := fmt.Sprintf("would install %s, signed by %q", p.path, sig.Chain[0].Name)
		return &base.Result{Label: label, Success: true, Changed: true, Message: &msg, Return: value}, nil
	}

	out, err := a.run("/usr/sbin/installer", "-pkg", pkgPath, "-target", p.target)
	if err != nil {
		return nil, fmt.Errorf("install_mac_pkg(label=%q): installer failed: %w\n%s", label, err, strings.TrimSpace(string(out)))
	}
	msg := fmt.Sprintf("installed %s, signed by %q", p.path, sig.Chain[0].Name)
	return &base.Result{Label: label, Success: true, Changed: true, Message: &msg, Return: value}, nil
}

var versionLine = regexp.MustCompile(`(?m)^version: (.+)$`)

// installedVersion is the version of package id installed on target, or ""
// when it isn't installed.
func (a *pkgAction) installedVersion(id, target string) (string, error) {
	out, err := a.run("/usr/sbin/pkgutil", "--pkg-info", id, "--volume", target)
	if err != nil {
		if strings.Contains(string(out), "No receipt for") {
			return "", nil
		}
		return "", fmt.Errorf("pkgutil --pkg-info %s: %w\n%s", id, err, strings.TrimSpace(string(out)))
	}
	m := versionLine.FindSubmatch(out)
	if m == nil {
		return "", fmt.Errorf("pkgutil --pkg-info %s: no version in output:\n%s", id, out)
	}
	return strings.TrimSpace(string(m[1])), nil
}

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

// copyFile copies src to a new file dst and returns the sha256 of what it
// wrote.
func copyFile(src, dst string) (string, error) {
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	h := starcmsha256.New()
	if _, err := io.Copy(io.MultiWriter(out, h), in); err != nil {
		out.Close()
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	return starcmsha256.Hex(h), nil
}

// Option configures install_mac_pkg().
type Option func(*pkgAction)

// WithStoreCheck makes install_mac_pkg() call check on the package path first,
// so a package from the local store is re-hashed before it's used.
func WithStoreCheck(check func(path string) error) Option {
	return func(a *pkgAction) { a.checkStored = check }
}

// WithGOOS replaces the operating system install_mac_pkg() thinks it runs on,
// for tests.
func WithGOOS(goos string) Option {
	return func(a *pkgAction) { a.goos = goos }
}

// New returns the install_mac_pkg() builtin, which runs pkgutil and
// installer with executor.
//
//	install_mac_pkg("agent.pkg", signer = "Developer ID Installer: Example Corp (ABCDE12345)", label = "install agent")
func New(ctx context.Context, executor shelllib.Executor, opts ...Option) *base.Module {
	a := &pkgAction{executor: executor, goos: runtime.GOOS}
	for _, opt := range opts {
		opt(a)
	}
	return base.NewModule(
		ctx,
		"install_mac_pkg",
		[]base.ArgPair{
			base.Arg[string]("path"),
			base.Arg[string]("signer"),
			base.OptionalArg[string]("sha256"),
			base.OptionalArgAllowNone[bool]("require_notarized"),
			base.OptionalArg[string]("target"),
			base.OptionalArg[string]("id"),
			base.OptionalArg[string]("version"),
		},
		a,
	)
}
