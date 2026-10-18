// Package macpkg implements install_mac_pkg(), which installs a macOS .pkg only
// after checking that it's signed by the certificate the configuration
// pins.
package macpkg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/discentem/starcm/functions/base"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Runner runs a command and returns its combined stdout and stderr. err is
// non-nil when the command couldn't start or exited non-zero.
type Runner func(ctx context.Context, name string, args ...string) ([]byte, error)

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

type pkgAction struct {
	run  Runner
	goos string
	// checkStored verifies path if it names a file in the local store.
	// Nil means no store.
	checkStored func(path string) error
}

var _ base.Runnable = (*pkgAction)(nil)

type parsedArgs struct {
	path             string
	signerSHA256     string
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
	signer, err := starlarkhelpers.FindValueinKwargs(kwargs, "signer_sha256")
	if err != nil {
		return nil, err
	}
	if p.signerSHA256, err = NormalizeFingerprint(*signer); err != nil {
		return nil, fmt.Errorf("install_mac_pkg(label=%q): signer_sha256: %w", label, err)
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
		installed, err := a.installedVersion(ctx, p.id, p.target)
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
	if err := copyFile(p.path, pkgPath); err != nil {
		return nil, fmt.Errorf("install_mac_pkg(label=%q): %w", label, err)
	}

	sig, err := a.checkSignature(ctx, pkgPath)
	if err != nil {
		return nil, fmt.Errorf("install_mac_pkg(label=%q): %s: %w", label, p.path, err)
	}
	if err := sig.Check(p.signerSHA256, p.requireNotarized); err != nil {
		return nil, fmt.Errorf("install_mac_pkg(label=%q): %s: %w", label, p.path, err)
	}
	value := sig.Value()

	if p.whatIf {
		msg := fmt.Sprintf("would install %s, signed by %q", p.path, sig.Chain[0].Name)
		return &base.Result{Label: label, Success: true, Changed: true, Message: &msg, Return: value}, nil
	}

	out, err := a.run(ctx, "/usr/sbin/installer", "-pkg", pkgPath, "-target", p.target)
	if err != nil {
		return nil, fmt.Errorf("install_mac_pkg(label=%q): installer failed: %w\n%s", label, err, strings.TrimSpace(string(out)))
	}
	msg := fmt.Sprintf("installed %s, signed by %q", p.path, sig.Chain[0].Name)
	return &base.Result{Label: label, Success: true, Changed: true, Message: &msg, Return: value}, nil
}

var versionLine = regexp.MustCompile(`(?m)^version: (.+)$`)

// installedVersion is the version of package id installed on target, or ""
// when it isn't installed.
func (a *pkgAction) installedVersion(ctx context.Context, id, target string) (string, error) {
	out, err := a.run(ctx, "/usr/sbin/pkgutil", "--pkg-info", id, "--volume", target)
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

func (a *pkgAction) checkSignature(ctx context.Context, path string) (*Signature, error) {
	// pkgutil exits non-zero for unsigned packages; the output says why.
	out, runErr := a.run(ctx, "/usr/sbin/pkgutil", "--check-signature", path)
	sig, err := ParseCheckSignature(string(out))
	if err != nil {
		if runErr != nil {
			return nil, fmt.Errorf("pkgutil --check-signature: %w\n%s", runErr, strings.TrimSpace(string(out)))
		}
		return nil, err
	}
	return sig, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// Certificate is one entry of a signature's certificate chain.
type Certificate struct {
	Name   string
	SHA256 string // lowercase hex, no separators
}

// Signature is what `pkgutil --check-signature` reports.
type Signature struct {
	Status    string
	Notarized bool
	Chain     []Certificate // leaf first
}

// trustedStatuses are the pkgutil statuses of signatures macOS trusts for
// installation. Anything else (no signature, an untrusted or expired
// certificate, a development certificate) is refused.
var trustedStatuses = []string{
	"signed Apple Software",
	"signed by a developer certificate issued by Apple for distribution",
	"signed by a certificate trusted by macOS",
	"signed by a certificate trusted by Mac OS X",
}

var (
	statusLine       = regexp.MustCompile(`^\s*Status: (.+?)\s*$`)
	notarizationLine = regexp.MustCompile(`^\s*Notarization: (.+?)\s*$`)
	chainEntry       = regexp.MustCompile(`^\s*\d+\. (.+?)\s*$`)
	hexLine          = regexp.MustCompile(`^\s*([0-9A-Fa-f]{2}( [0-9A-Fa-f]{2})*)\s*$`)
)

// ParseCheckSignature parses the output of `pkgutil --check-signature`.
func ParseCheckSignature(out string) (*Signature, error) {
	sig := &Signature{}
	var cur *Certificate
	inFingerprint := false
	for _, line := range strings.Split(out, "\n") {
		switch {
		case statusLine.MatchString(line) && sig.Status == "":
			sig.Status = statusLine.FindStringSubmatch(line)[1]
		case notarizationLine.MatchString(line):
			sig.Notarized = strings.Contains(line, "trusted by the Apple notary service")
		case chainEntry.MatchString(line):
			sig.Chain = append(sig.Chain, Certificate{Name: chainEntry.FindStringSubmatch(line)[1]})
			cur = &sig.Chain[len(sig.Chain)-1]
			inFingerprint = false
		case strings.TrimSpace(line) == "SHA256 Fingerprint:" && cur != nil:
			inFingerprint = true
		case inFingerprint && hexLine.MatchString(line):
			cur.SHA256 += strings.ToLower(strings.ReplaceAll(strings.TrimSpace(line), " ", ""))
		default:
			inFingerprint = false
		}
	}
	if sig.Status == "" {
		return nil, fmt.Errorf("can't read pkgutil --check-signature output:\n%s", strings.TrimSpace(out))
	}
	return sig, nil
}

// Check returns an error unless the signature is trusted, its leaf
// certificate has the fingerprint want, and it's notarized when
// requireNotarized is set.
func (s *Signature) Check(want string, requireNotarized bool) error {
	trusted := false
	for _, t := range trustedStatuses {
		if s.Status == t {
			trusted = true
		}
	}
	if !trusted {
		return fmt.Errorf("signature is not trusted: %s", s.Status)
	}
	if len(s.Chain) == 0 || len(s.Chain[0].SHA256) != 64 {
		return errors.New("pkgutil reported no signing certificate fingerprint")
	}
	if got := s.Chain[0].SHA256; got != want {
		return fmt.Errorf("signed by %q with certificate sha256 %s, not the expected %s", s.Chain[0].Name, got, want)
	}
	if requireNotarized && !s.Notarized {
		return errors.New("not notarized by Apple; pass require_notarized = False to install it anyway")
	}
	return nil
}

var teamID = regexp.MustCompile(`\(([A-Z0-9]{10})\)$`)

// Value is install_mac_pkg()'s result value: who signed the package.
func (s *Signature) Value() starlark.Value {
	leaf := s.Chain[0]
	team := ""
	if m := teamID.FindStringSubmatch(leaf.Name); m != nil {
		team = m[1]
	}
	return starlarkstruct.FromStringDict(starlark.String("pkg_signature"), starlark.StringDict{
		"signer":        starlark.String(leaf.Name),
		"signer_sha256": starlark.String(leaf.SHA256),
		"team_id":       starlark.String(team),
		"notarized":     starlark.Bool(s.Notarized),
	})
}

var nonHex = regexp.MustCompile(`[\s:]`)

// NormalizeFingerprint accepts a sha256 fingerprint as pkgutil prints it
// ("1E 77 DE ..."), with colons, or as plain hex, and returns lowercase hex.
func NormalizeFingerprint(s string) (string, error) {
	h := strings.ToLower(nonHex.ReplaceAllString(s, ""))
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(h) {
		return "", fmt.Errorf("%q is not a sha256 fingerprint (64 hex digits)", s)
	}
	return h, nil
}

// Option configures install_mac_pkg().
type Option func(*pkgAction)

// WithStoreCheck makes install_mac_pkg() call check on the package path first,
// so a package from the local store is re-hashed before it's used.
func WithStoreCheck(check func(path string) error) Option {
	return func(a *pkgAction) { a.checkStored = check }
}

// WithRunner replaces how commands are run, for tests.
func WithRunner(run Runner) Option {
	return func(a *pkgAction) { a.run = run }
}

// WithGOOS replaces the operating system install_mac_pkg() thinks it runs on,
// for tests.
func WithGOOS(goos string) Option {
	return func(a *pkgAction) { a.goos = goos }
}

// New returns the install_mac_pkg() builtin.
//
//	install_mac_pkg("agent.pkg", signer_sha256 = "1E 77 DE ...", label = "install agent")
func New(ctx context.Context, opts ...Option) *base.Module {
	a := &pkgAction{run: runCommand, goos: runtime.GOOS}
	for _, opt := range opts {
		opt(a)
	}
	return base.NewModule(
		ctx,
		"install_mac_pkg",
		[]base.ArgPair{
			base.Arg[string]("path"),
			base.Arg[string]("signer_sha256"),
			base.Arg[bool]("require_notarized??"),
			base.Arg[string]("target?"),
			base.Arg[string]("id?"),
			base.Arg[string]("version?"),
		},
		a,
	)
}
