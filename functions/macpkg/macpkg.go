// Package macpkg implements install_mac_pkg(), which installs a macOS .pkg
// only after checking that it's signed, through Apple, by the developer
// certificate the configuration names.
package macpkg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/libraries/fileutils"
	"github.com/discentem/starcm/libraries/macsig"
	shelllib "github.com/discentem/starcm/libraries/shell"
)

type pkgAction struct {
	executor shelllib.Executor
	goos     string
	// checkStored verifies path if it names a file in the local store.
	// Nil means no store.
	checkStored func(path string) error
}

var _ base.Action = (*pkgAction)(nil)

type parsedArgs struct {
	// fn names the builtin in errors, e.g. install_mac_pkg.
	fn, label string
	path      string
	// name is how messages and errors name the package: its path, or the
	// URL it was fetched from.
	name             string
	signer           string
	sha256           string // "" when not pinned
	requireNotarized bool
	target           string
	id, version      string
	whatIf           bool
}

func (p *parsedArgs) errorf(format string, args ...any) error {
	return fmt.Errorf("%s(label=%q): "+format, append([]any{p.fn, p.label}, args...)...)
}

// parseArgs reads the arguments install_mac_pkg() and
// install_remote_mac_pkg() share, except paths, which are only checked once
// we know we're on macOS (see checkPlatform).
func (a *pkgAction) parseArgs(inst *base.FunctionInstance, fn string) (*parsedArgs, error) {
	p := &parsedArgs{fn: fn, label: inst.Label, whatIf: inst.WhatIf}
	signer, _ := base.Get[string](inst, "signer")
	if p.signer = strings.TrimSpace(signer); p.signer == "" {
		return nil, p.errorf("signer must name the certificate, e.g. \"Developer ID Installer: Example Corp (ABCDE12345)\"")
	}
	sum, _ := base.Get[string](inst, "sha256")
	if p.sha256 = strings.ToLower(sum); p.sha256 != "" && !sha256Hex.MatchString(p.sha256) {
		return nil, p.errorf("sha256 %q is not 64 hex characters", sum)
	}
	p.requireNotarized = true
	if v, ok := base.Get[bool](inst, "require_notarized"); ok {
		p.requireNotarized = v
	}
	p.id, _ = base.Get[string](inst, "id")
	p.version, _ = base.Get[string](inst, "version")
	if (p.id == "") != (p.version == "") {
		return nil, p.errorf("id and version must be given together")
	}
	return p, nil
}

// checkPlatform refuses to run anywhere but macOS, then reads target.
// Whether a path is absolute depends on the OS, so paths are checked here.
func (a *pkgAction) checkPlatform(inst *base.FunctionInstance, p *parsedArgs) error {
	if a.goos != "darwin" {
		return p.errorf("%s() only works on macOS", p.fn)
	}
	target, given, err := inst.AbsPath("target")
	if err != nil {
		return err
	}
	if !given {
		target = "/"
	}
	p.target = target
	return nil
}

// alreadyInstalled returns a result when p's id and version are already
// installed on its target, or nil when the package should be installed.
func (a *pkgAction) alreadyInstalled(p *parsedArgs) (*base.Result, error) {
	if p.id == "" {
		return nil, nil
	}
	installed, err := a.installedVersion(p.id, p.target)
	if err != nil {
		return nil, p.errorf("%w", err)
	}
	if installed != p.version {
		return nil, nil
	}
	msg := fmt.Sprintf("%s %s is already installed", p.id, p.version)
	return &base.Result{Label: p.label, Success: true, Message: &msg}, nil
}

func (a *pkgAction) Execute(inst *base.FunctionInstance) (*base.Result, error) {
	p, err := a.parseArgs(inst, "install_mac_pkg")
	if err != nil {
		return nil, err
	}
	if err := a.checkPlatform(inst, p); err != nil {
		return nil, err
	}
	if p.path, _, err = inst.AbsPath("path"); err != nil {
		return nil, err
	}
	p.name = p.path
	if r, err := a.alreadyInstalled(p); r != nil || err != nil {
		return r, err
	}
	return a.install(p)
}

// install checks the package at p.path and installs it.
func (a *pkgAction) install(p *parsedArgs) (*base.Result, error) {
	if a.checkStored != nil {
		if err := a.checkStored(p.path); err != nil {
			return nil, p.errorf("%w", err)
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
	sum, err := fileutils.CopyNew(p.path, pkgPath, 0o600)
	if err != nil {
		return nil, p.errorf("%w", err)
	}
	if p.sha256 != "" && sum != p.sha256 {
		return nil, p.errorf("%s has sha256 %s, not the expected %s", p.name, sum, p.sha256)
	}

	sig, err := macsig.New(a.executor).ReadPkg(pkgPath)
	if err != nil {
		return nil, p.errorf("%s: %w", p.name, err)
	}
	if err := sig.Check(p.signer, p.requireNotarized); err != nil {
		return nil, p.errorf("%s: %w", p.name, err)
	}
	value := sig.Value()

	if p.whatIf {
		msg := fmt.Sprintf("would install %s, signed by %q", p.name, sig.Chain[0].Name)
		return &base.Result{Label: p.label, Success: true, Changed: true, Message: &msg, Return: value}, nil
	}

	out, err := shelllib.CombinedOutput(a.executor, "/usr/sbin/installer", "-pkg", pkgPath, "-target", p.target)
	if err != nil {
		return nil, p.errorf("installer failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	msg := fmt.Sprintf("installed %s, signed by %q", p.name, sig.Chain[0].Name)
	return &base.Result{Label: p.label, Success: true, Changed: true, Message: &msg, Return: value}, nil
}

var versionLine = regexp.MustCompile(`(?m)^version: (.+)$`)

// installedVersion is the version of package id installed on target, or ""
// when it isn't installed.
func (a *pkgAction) installedVersion(id, target string) (string, error) {
	out, err := shelllib.CombinedOutput(a.executor, "/usr/sbin/pkgutil", "--pkg-info", id, "--volume", target)
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

// pkgArgs are install_mac_pkg()'s own arguments.
var pkgArgs = []base.ArgPair{
	base.Arg[string]("path"),
	base.Arg[string]("signer"),
	base.OptionalArg[string]("sha256"),
	base.OptionalArgAllowNone[bool]("require_notarized"),
	base.OptionalArg[string]("target"),
	base.OptionalArg[string]("id"),
	base.OptionalArg[string]("version"),
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
//	install_mac_pkg(agent.value, signer = "Developer ID Installer: Example Corp (ABCDE12345)", label = "install agent")
func New(ctx context.Context, executor shelllib.Executor, opts ...Option) *base.Module {
	a := &pkgAction{executor: executor, goos: runtime.GOOS}
	for _, opt := range opts {
		opt(a)
	}
	return base.NewModule(
		ctx,
		"install_mac_pkg",
		pkgArgs,
		a,
	)
}
