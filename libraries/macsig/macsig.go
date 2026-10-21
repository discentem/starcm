// Package macsig reads and checks macOS code signatures: the signature of an
// installer package (.pkg) with pkgutil, and of a Mach-O binary or .app bundle
// with codesign. install_mac_pkg(), check_mac_signature() and
// file(source = ..., signer = ...) share it, so they accept and refuse
// exactly the same signatures.
package macsig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	starcmsha256 "github.com/discentem/starcm/libraries/sha256"
	shelllib "github.com/discentem/starcm/libraries/shell"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Certificate is one entry of a signature's certificate chain.
type Certificate struct {
	Name   string
	SHA256 string // lowercase hex, no separators
}

// Signature is what `pkgutil --check-signature` reports for a package, or
// what ReadCode works out for code in the same terms.
type Signature struct {
	Status    string
	Notarized bool
	Chain     []Certificate // leaf first
}

// Reader runs pkgutil and codesign with an executor, so tests can answer
// for them.
type Reader struct {
	Executor shelllib.Executor
}

// New returns a Reader that runs commands with executor.
func New(executor shelllib.Executor) *Reader {
	return &Reader{Executor: executor}
}

// ReadPkg reads the signature of the installer package at path.
func ReadPkg(path string) (*Signature, error) {
	return New(&shelllib.RealExecutor{}).ReadPkg(path)
}

// ReadCode reads the signature of the Mach-O binary or .app bundle at path.
func ReadCode(path string) (*Signature, error) {
	return New(&shelllib.RealExecutor{}).ReadCode(path)
}

// run runs a command and returns its combined stdout and stderr. err is
// non-nil when the command couldn't start or exited non-zero.
func (r *Reader) run(name string, args ...string) ([]byte, error) {
	r.Executor.Command(name, args...)
	return r.Executor.CombinedOutput()
}

// ReadPkg reads the signature of the installer package at path.
func (r *Reader) ReadPkg(path string) (*Signature, error) {
	// pkgutil exits non-zero for unsigned packages; the output says why.
	out, runErr := r.run("/usr/sbin/pkgutil", "--check-signature", path)
	sig, err := ParseCheckSignature(string(out))
	if err != nil {
		if runErr != nil {
			return nil, fmt.Errorf("pkgutil --check-signature: %w\n%s", runErr, strings.TrimSpace(string(out)))
		}
		return nil, err
	}
	return sig, nil
}

// The kinds of file Read reads.
const (
	KindAuto = "auto" // KindPkg for a .pkg, KindCode otherwise
	KindPkg  = "pkg"  // an installer package, read with ReadPkg
	KindCode = "code" // a Mach-O binary or .app bundle, read with ReadCode
)

// ResolveKind turns KindAuto into KindPkg or KindCode for path, and
// returns an error for an unknown kind.
func ResolveKind(path, kind string) (string, error) {
	switch kind {
	case KindPkg, KindCode:
		return kind, nil
	case KindAuto, "":
		if strings.EqualFold(filepath.Ext(path), ".pkg") {
			return KindPkg, nil
		}
		return KindCode, nil
	}
	return "", fmt.Errorf("kind must be %q, %q or %q, not %q", KindAuto, KindPkg, KindCode, kind)
}

// Read reads the signature of path as kind (see ResolveKind).
func (r *Reader) Read(path, kind string) (*Signature, error) {
	kind, err := ResolveKind(path, kind)
	if err != nil {
		return nil, err
	}
	if kind == KindPkg {
		return r.ReadPkg(path)
	}
	return r.ReadCode(path)
}

// codesign is the codesign binary ReadCode runs.
const codesign = "/usr/bin/codesign"

// The statuses ReadCode reports for code that Check refuses.
const (
	StatusUnsigned = "no signature"
	StatusAdHoc    = "ad-hoc signature"
	StatusOther    = "signed by a certificate that isn't an Apple or Developer ID Application certificate"
)

// ReadCode reads the signature of the Mach-O binary or .app bundle at path
// and reports it in pkgutil's terms, so Check treats code and packages
// alike:
//
//   - `codesign --verify --strict` checks the code hasn't been modified
//     since it was signed. Unsigned code reports StatusUnsigned; any other
//     failure, such as a modified file, is an error.
//   - `codesign -dvvv` names the certificate chain (its Authority= lines,
//     leaf first), and `codesign -d --extract-certificates` writes each
//     certificate out so its sha256 can be taken.
//   - The status comes from a code requirement, which the Security
//     framework checks against the certificates themselves rather than
//     their printed names (see StatusFor). Plain `codesign --verify` only
//     checks the code against its own designated requirement, which whoever
//     signed it chose, so names alone could be forged with a private CA.
//   - Notarization is the "notarized" requirement, checked online with
//     --check-notarization. spctl can't be used: it refuses to assess a
//     command-line binary ("does not seem to be an app").
func (r *Reader) ReadCode(path string) (*Signature, error) {
	verify := []string{"--verify", "--strict"}
	if info, err := os.Stat(path); err != nil {
		return nil, err
	} else if info.IsDir() {
		verify = append(verify, "--deep")
	}
	out, err := r.run(codesign, append(verify, path)...)
	if err != nil {
		if strings.Contains(string(out), "code object is not signed at all") {
			return &Signature{Status: StatusUnsigned}, nil
		}
		return nil, fmt.Errorf("codesign --verify: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	out, err = r.run(codesign, "-dvvv", path)
	if err != nil {
		return nil, fmt.Errorf("codesign -dvvv: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	d := ParseDisplay(string(out))
	if d.AdHoc {
		return &Signature{Status: StatusAdHoc}, nil
	}
	if len(d.Authorities) == 0 {
		return nil, fmt.Errorf("can't read codesign -dvvv output:\n%s", strings.TrimSpace(string(out)))
	}

	sums, err := r.certificateSHA256s(path)
	if err != nil {
		return nil, err
	}
	if len(sums) != len(d.Authorities) {
		return nil, fmt.Errorf("codesign named %d certificates but extracted %d", len(d.Authorities), len(sums))
	}
	sig := &Signature{Status: StatusOther}
	for i, name := range d.Authorities {
		sig.Chain = append(sig.Chain, Certificate{Name: name, SHA256: sums[i]})
	}

	for _, status := range []string{StatusDeveloperID, StatusAppleSoftware} {
		req, ok := Requirement(status, d.Authorities[0], d.TeamID)
		if !ok {
			continue
		}
		satisfied, err := r.satisfies(path, req)
		if err != nil {
			return nil, err
		}
		if satisfied {
			sig.Status = status
			break
		}
	}
	if sig.Status == StatusDeveloperID {
		if sig.Notarized, err = r.satisfies(path, "notarized", "--check-notarization"); err != nil {
			return nil, err
		}
	}
	return sig, nil
}

// satisfies reports whether the code at path satisfies requirement, in
// Apple's code requirement language.
func (r *Reader) satisfies(path, requirement string, flags ...string) (bool, error) {
	args := append([]string{"--verify", "--strict"}, flags...)
	// "-R=<text>": an argument starting with "=" is requirement source text
	// rather than a file (man codesign, SPECIFYING REQUIREMENTS).
	args = append(args, "-R="+requirement, path)
	out, err := r.run(codesign, args...)
	switch {
	case err == nil:
		return true, nil
	case strings.Contains(string(out), "failed to satisfy specified code requirement"):
		return false, nil
	default:
		return false, fmt.Errorf("codesign --verify -R=%q: %w\n%s", requirement, err, strings.TrimSpace(string(out)))
	}
}

// certificateSHA256s returns the sha256 of each certificate in the code's
// signature, leaf first: the same fingerprints pkgutil prints.
func (r *Reader) certificateSHA256s(path string) ([]string, error) {
	dir, err := os.MkdirTemp("", "starcm-macsig-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	prefix := filepath.Join(dir, "cert")
	if out, err := r.run(codesign, "-d", "--extract-certificates="+prefix, path); err != nil {
		return nil, fmt.Errorf("codesign --extract-certificates: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	var sums []string
	for i := 0; ; i++ {
		f, err := os.Open(prefix + strconv.Itoa(i))
		if errors.Is(err, os.ErrNotExist) {
			return sums, nil
		} else if err != nil {
			return nil, err
		}
		sum, err := starcmsha256.FromReader(f)
		f.Close()
		if err != nil {
			return nil, err
		}
		sums = append(sums, sum)
	}
}

// OIDs Apple puts in the certificates it issues, from TN3127, "Inside Code
// Signing: Requirements".
const (
	// oidDeveloperIDCA marks the Developer ID Certification Authority.
	oidDeveloperIDCA = "1.2.840.113635.100.6.2.6"
	// oidDeveloperIDApplication marks a Developer ID Application certificate.
	// (Developer ID Installer certificates carry 1.2.840.113635.100.6.1.14
	// instead, and can't sign code.)
	oidDeveloperIDApplication = "1.2.840.113635.100.6.1.13"
)

// Requirement is the code requirement code signed by leaf must satisfy to
// have status: StatusDeveloperID or StatusAppleSoftware. ok is false for any
// other status, or a name the requirement language can't quote.
//
// "anchor apple generic" means the chain ends at Apple's root, "anchor
// apple" that the code is Apple's own (Code Signing Requirement Language,
// Code Signing Guide). The OID checks are the ones in Xcode's designated
// requirement for Developer ID code (TN3127); they only mean something
// together with "anchor apple generic". Unlike the designated requirement,
// these also pin the leaf certificate's common name, the signer Check
// compares, and its organizational unit, the Team ID.
func Requirement(status, leaf, teamID string) (req string, ok bool) {
	if strings.ContainsAny(leaf+teamID, `"\`) {
		return "", false
	}
	switch status {
	case StatusDeveloperID:
		if teamID == "" {
			return "", false
		}
		return fmt.Sprintf(`anchor apple generic`+
			` and certificate 1[field.%s] exists`+
			` and certificate leaf[field.%s] exists`+
			` and certificate leaf[subject.CN] = "%s"`+
			` and certificate leaf[subject.OU] = "%s"`,
			oidDeveloperIDCA, oidDeveloperIDApplication, leaf, teamID), true
	case StatusAppleSoftware:
		return fmt.Sprintf(`anchor apple and certificate leaf[subject.CN] = "%s"`, leaf), true
	}
	return "", false
}

// Display is what ReadCode needs from `codesign -dvvv`.
type Display struct {
	Authorities []string // the certificate chain's names, leaf first
	TeamID      string   // "" when not set
	AdHoc       bool
}

// ParseDisplay parses the output of `codesign -dvvv`.
func ParseDisplay(out string) Display {
	var d Display
	for _, line := range strings.Split(out, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch key {
		case "Authority":
			d.Authorities = append(d.Authorities, value)
		case "TeamIdentifier":
			if value != "not set" {
				d.TeamID = value
			}
		case "Signature":
			d.AdHoc = value == "adhoc"
		}
	}
	return d
}

// appleStatuses are the pkgutil statuses of signatures whose certificate
// Apple issued for distribution. The signer is matched by name, and Apple
// only issues a Developer ID certificate to the team it names, so anything
// else is refused: no signature, an untrusted, expired or development
// certificate, and certificates from other CAs that macOS trusts ("signed by
// a certificate trusted by macOS"), which could carry any name.
var appleStatuses = []string{
	StatusAppleSoftware,
	StatusDeveloperID,
}

// The statuses Check accepts, as pkgutil prints them. ReadCode reports code
// signatures in the same words.
const (
	StatusAppleSoftware = "signed Apple Software"
	StatusDeveloperID   = "signed by a developer certificate issued by Apple for distribution"
)

// appleRoot is the name pkgutil prints for the last certificate of every
// chain Apple issues.
const appleRoot = "Apple Root CA"

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

// Check returns an error unless the signature chains to Apple, its leaf
// certificate is named signer exactly, and it's notarized when
// requireNotarized is set.
func (s *Signature) Check(signer string, requireNotarized bool) error {
	trusted := false
	for _, t := range appleStatuses {
		if s.Status == t {
			trusted = true
		}
	}
	if !trusted {
		return fmt.Errorf("not signed with a certificate Apple issued for distribution: %s", s.Status)
	}
	if len(s.Chain) < 2 || s.Chain[len(s.Chain)-1].Name != appleRoot {
		return fmt.Errorf("certificate chain doesn't end at %s", appleRoot)
	}
	if got := s.Chain[0].Name; got != signer {
		return fmt.Errorf("signed by %q, not the expected %q", got, signer)
	}
	if requireNotarized && !s.Notarized {
		return errors.New("not notarized by Apple; pass require_notarized = False to install it anyway")
	}
	return nil
}

var teamID = regexp.MustCompile(`\(([A-Z0-9]{10})\)$`)

// Fields are the fields of Value's struct: who signed it.
func (s *Signature) Fields() starlark.StringDict {
	leaf := s.Chain[0]
	team := ""
	if m := teamID.FindStringSubmatch(leaf.Name); m != nil {
		team = m[1]
	}
	return starlark.StringDict{
		"signer":        starlark.String(leaf.Name),
		"signer_sha256": starlark.String(leaf.SHA256),
		"team_id":       starlark.String(team),
		"notarized":     starlark.Bool(s.Notarized),
	}
}

// Value is install_mac_pkg()'s result value: who signed the package. Call it
// only after Check succeeds.
func (s *Signature) Value() starlark.Value {
	return starlarkstruct.FromStringDict(starlark.String("pkg_signature"), s.Fields())
}
