package macsig

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	shelllib "github.com/discentem/starcm/libraries/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tailscaleSigner and tailscaleSHA256 are the installer certificate in
// testdata/developer-id-notarized.txt, real `pkgutil --check-signature`
// output for a Developer ID signed, notarized package.
const (
	tailscaleSigner = "Developer ID Installer: Tailscale Inc. (W5364U7YZB)"
	tailscaleSHA256 = "1e77de6fe47d426984c5aa180d581f3246197f73c0c713277b844008b936e093"
)

func readTestdata(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return string(b)
}

func TestParseCheckSignature(t *testing.T) {
	sig, err := ParseCheckSignature(readTestdata(t, "developer-id-notarized.txt"))
	require.NoError(t, err)
	assert.Equal(t, "signed by a developer certificate issued by Apple for distribution", sig.Status)
	assert.True(t, sig.Notarized)
	require.Len(t, sig.Chain, 3)
	assert.Equal(t, "Developer ID Installer: Tailscale Inc. (W5364U7YZB)", sig.Chain[0].Name)
	assert.Equal(t, tailscaleSHA256, sig.Chain[0].SHA256)
	assert.Equal(t, "Apple Root CA", sig.Chain[2].Name)
	assert.Equal(t, "b0b1730ecbc7ff4505142c49f1295e6eda6bcaed7e2c68c5be91b5a11001f024", sig.Chain[2].SHA256)

	unsigned, err := ParseCheckSignature(readTestdata(t, "unsigned.txt"))
	require.NoError(t, err)
	assert.Equal(t, "no signature", unsigned.Status)
	assert.Empty(t, unsigned.Chain)

	_, err = ParseCheckSignature("garbage")
	assert.Error(t, err)
}

func TestSignatureCheck(t *testing.T) {
	sig, err := ParseCheckSignature(readTestdata(t, "developer-id-notarized.txt"))
	require.NoError(t, err)
	assert.NoError(t, sig.Check(tailscaleSigner, true))

	// Another developer's certificate, or a near miss, is refused.
	for _, other := range []string{
		"Developer ID Installer: Some Other Co (ABCDE12345)",
		"Developer ID Installer: Tailscale Inc. (ABCDE12345)",
		"Developer ID Installer: Tailscale Inc.",
		"W5364U7YZB",
		"developer id installer: tailscale inc. (w5364u7yzb)",
	} {
		assert.ErrorContains(t, sig.Check(other, true), `signed by "`+tailscaleSigner+`", not the expected "`+other+`"`)
	}

	// Naming an intermediate or the root isn't enough: only the leaf counts.
	assert.Error(t, sig.Check("Developer ID Certification Authority", true))
	assert.Error(t, sig.Check("Apple Root CA", true))

	notNotarized := *sig
	notNotarized.Notarized = false
	assert.ErrorContains(t, notNotarized.Check(tailscaleSigner, true), "not notarized")
	assert.NoError(t, notNotarized.Check(tailscaleSigner, false))

	// Only certificates Apple issued count, even if macOS trusts the
	// certificate: a private CA could issue one with any name.
	for _, status := range []string{
		"no signature",
		"signed by untrusted certificate",
		"signed by a certificate that has since expired",
		"signed by a developer certificate issued by Apple (Development)",
		"signed by a certificate trusted by macOS",
	} {
		s := *sig
		s.Status = status
		assert.ErrorContains(t, s.Check(tailscaleSigner, false), "not signed with a certificate Apple issued for distribution: "+status)
	}

	// A chain that doesn't end at Apple's root is refused.
	forged := *sig
	forged.Chain = append([]Certificate{}, sig.Chain[:2]...)
	assert.ErrorContains(t, forged.Check(tailscaleSigner, false), "certificate chain doesn't end at Apple Root CA")
}

// hashicorpSigner is the leaf certificate in testdata/codesign-developer-id.txt,
// real `codesign -dvvv` output for Terraform 1.17.0 (darwin_arm64).
const hashicorpSigner = "Developer ID Application: Hashicorp, Inc. (D38WU7D763)"

func TestParseDisplay(t *testing.T) {
	d := ParseDisplay(readTestdata(t, "codesign-developer-id.txt"))
	assert.Equal(t, []string{hashicorpSigner, "Developer ID Certification Authority", "Apple Root CA"}, d.Authorities)
	assert.Equal(t, "D38WU7D763", d.TeamID)
	assert.False(t, d.AdHoc)

	apple := ParseDisplay(readTestdata(t, "codesign-apple.txt"))
	assert.Equal(t, []string{"macOS Software Signing", "Apple Code Signing Certification Authority", "Apple Root CA"}, apple.Authorities)
	assert.Empty(t, apple.TeamID)

	adhoc := ParseDisplay(readTestdata(t, "codesign-adhoc.txt"))
	assert.True(t, adhoc.AdHoc)
	assert.Empty(t, adhoc.Authorities)
	assert.Empty(t, adhoc.TeamID)
}

func TestRequirement(t *testing.T) {
	req, ok := Requirement(StatusDeveloperID, hashicorpSigner, "D38WU7D763")
	require.True(t, ok)
	assert.Equal(t, `anchor apple generic`+
		` and certificate 1[field.1.2.840.113635.100.6.2.6] exists`+
		` and certificate leaf[field.1.2.840.113635.100.6.1.13] exists`+
		` and certificate leaf[subject.CN] = "Developer ID Application: Hashicorp, Inc. (D38WU7D763)"`+
		` and certificate leaf[subject.OU] = "D38WU7D763"`, req)

	req, ok = Requirement(StatusAppleSoftware, "macOS Software Signing", "")
	require.True(t, ok)
	assert.Equal(t, `anchor apple and certificate leaf[subject.CN] = "macOS Software Signing"`, req)

	// Developer ID needs a Team ID, names must be quotable, and no other
	// status has a requirement.
	for _, c := range [][3]string{
		{StatusDeveloperID, hashicorpSigner, ""},
		{StatusDeveloperID, `Developer ID Application: A" or anchor trusted or "`, "D38WU7D763"},
		{StatusDeveloperID, `Developer ID Application: A\`, "D38WU7D763"},
		{StatusDeveloperID, hashicorpSigner, `D38WU7D763" or "`},
		{StatusOther, hashicorpSigner, "D38WU7D763"},
	} {
		_, ok := Requirement(c[0], c[1], c[2])
		assert.False(t, ok, "%q", c)
	}
}

// fakeCodesign is a shelllib.Executor that answers codesign the way macOS
// does for a signature with the given -dvvv output and certificates.
type fakeCodesign struct {
	pending []string
	calls   [][]string
	// verify is codesign --verify's output; non-empty means it fails.
	verify string
	// display is codesign -dvvv's output.
	display string
	// certs are the DER certificates --extract-certificates writes.
	certs []string
	// satisfied says which -R requirements pass.
	satisfied map[string]bool
}

var _ shelllib.Executor = (*fakeCodesign)(nil)

func (f *fakeCodesign) Command(name string, args ...string) {
	f.pending = append([]string{name}, args...)
}
func (f *fakeCodesign) Stream(...io.WriteCloser) error { return errors.New("not used") }
func (f *fakeCodesign) ExitCode() (int, error)         { return 0, errors.New("not used") }

func (f *fakeCodesign) CombinedOutput() ([]byte, error) {
	f.calls = append(f.calls, f.pending)
	args := f.pending[1:]
	if f.pending[0] != "/usr/bin/codesign" {
		return nil, errors.New("unexpected command " + f.pending[0])
	}
	switch {
	case args[0] == "--verify":
		for _, a := range args {
			if req, ok := strings.CutPrefix(a, "-R="); ok {
				if f.satisfied[req] {
					return nil, nil
				}
				return []byte("test-requirement: code failed to satisfy specified code requirement(s)\n"), errors.New("exit status 3")
			}
		}
		if f.verify != "" {
			return []byte(f.verify), errors.New("exit status 1")
		}
		return nil, nil
	case args[0] == "-dvvv":
		return []byte(f.display), nil
	case args[0] == "-d" && strings.HasPrefix(args[1], "--extract-certificates="):
		prefix := strings.TrimPrefix(args[1], "--extract-certificates=")
		for i, c := range f.certs {
			if err := os.WriteFile(prefix+strconv.Itoa(i), []byte(c), 0o600); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
	return nil, errors.New("unexpected codesign arguments")
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestReadCode(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "terraform")
	require.NoError(t, os.WriteFile(bin, []byte("binary"), 0o755))
	devID, _ := Requirement(StatusDeveloperID, hashicorpSigner, "D38WU7D763")
	certs := []string{"leaf", "intermediate", "root"}

	t.Run("Developer ID, notarized", func(t *testing.T) {
		f := &fakeCodesign{
			display:   readTestdata(t, "codesign-developer-id.txt"),
			certs:     certs,
			satisfied: map[string]bool{devID: true, "notarized": true},
		}
		sig, err := New(f).ReadCode(bin)
		require.NoError(t, err)
		assert.Equal(t, StatusDeveloperID, sig.Status)
		assert.True(t, sig.Notarized)
		assert.Equal(t, []Certificate{
			{Name: hashicorpSigner, SHA256: sha256Hex("leaf")},
			{Name: "Developer ID Certification Authority", SHA256: sha256Hex("intermediate")},
			{Name: "Apple Root CA", SHA256: sha256Hex("root")},
		}, sig.Chain)
		assert.NoError(t, sig.Check(hashicorpSigner, true))
		assert.ErrorContains(t, sig.Check("Developer ID Application: Example Corp (ABCDE12345)", true),
			`signed by "`+hashicorpSigner+`", not the expected "Developer ID Application: Example Corp (ABCDE12345)"`)
		// Notarization is checked online.
		assert.Contains(t, f.calls, []string{"/usr/bin/codesign", "--verify", "--strict", "--check-notarization", "-R=notarized", bin})
	})

	t.Run("Developer ID, not notarized", func(t *testing.T) {
		f := &fakeCodesign{
			display:   readTestdata(t, "codesign-developer-id.txt"),
			certs:     certs,
			satisfied: map[string]bool{devID: true},
		}
		sig, err := New(f).ReadCode(bin)
		require.NoError(t, err)
		assert.False(t, sig.Notarized)
		assert.ErrorContains(t, sig.Check(hashicorpSigner, true), "not notarized")
		assert.NoError(t, sig.Check(hashicorpSigner, false))
	})

	t.Run("names that don't satisfy the requirement are refused", func(t *testing.T) {
		// A private CA can issue certificates named anything, including
		// "Apple Root CA": only the requirement check counts.
		f := &fakeCodesign{
			display:   readTestdata(t, "codesign-developer-id.txt"),
			certs:     certs,
			satisfied: map[string]bool{"notarized": true},
		}
		sig, err := New(f).ReadCode(bin)
		require.NoError(t, err)
		assert.Equal(t, StatusOther, sig.Status)
		assert.False(t, sig.Notarized)
		assert.ErrorContains(t, sig.Check(hashicorpSigner, false), "not signed with a certificate Apple issued for distribution: "+StatusOther)
	})

	t.Run("Apple's own code", func(t *testing.T) {
		apple, _ := Requirement(StatusAppleSoftware, "macOS Software Signing", "")
		f := &fakeCodesign{
			display:   readTestdata(t, "codesign-apple.txt"),
			certs:     certs,
			satisfied: map[string]bool{apple: true},
		}
		sig, err := New(f).ReadCode(bin)
		require.NoError(t, err)
		assert.Equal(t, StatusAppleSoftware, sig.Status)
		assert.False(t, sig.Notarized)
		assert.NoError(t, sig.Check("macOS Software Signing", false))
		assert.ErrorContains(t, sig.Check("macOS Software Signing", true), "not notarized")
	})

	t.Run("ad-hoc", func(t *testing.T) {
		f := &fakeCodesign{display: readTestdata(t, "codesign-adhoc.txt")}
		sig, err := New(f).ReadCode(bin)
		require.NoError(t, err)
		assert.Equal(t, StatusAdHoc, sig.Status)
		assert.ErrorContains(t, sig.Check(hashicorpSigner, false), "not signed with a certificate Apple issued for distribution: ad-hoc signature")
	})

	t.Run("unsigned", func(t *testing.T) {
		f := &fakeCodesign{verify: bin + ": code object is not signed at all\n"}
		sig, err := New(f).ReadCode(bin)
		require.NoError(t, err)
		assert.Equal(t, StatusUnsigned, sig.Status)
		assert.ErrorContains(t, sig.Check(hashicorpSigner, false), "not signed with a certificate Apple issued for distribution: no signature")
	})

	t.Run("modified since signing", func(t *testing.T) {
		f := &fakeCodesign{verify: bin + ": main executable failed strict validation\n"}
		_, err := New(f).ReadCode(bin)
		assert.ErrorContains(t, err, "codesign --verify: exit status 1\n"+bin+": main executable failed strict validation")
	})

	t.Run("a certificate missing from the extracted chain", func(t *testing.T) {
		f := &fakeCodesign{display: readTestdata(t, "codesign-developer-id.txt"), certs: certs[:2]}
		_, err := New(f).ReadCode(bin)
		assert.ErrorContains(t, err, "codesign named 3 certificates but extracted 2")
	})

	t.Run("bundles are verified deeply", func(t *testing.T) {
		app := t.TempDir()
		f := &fakeCodesign{verify: "x: code object is not signed at all"}
		_, err := New(f).ReadCode(app)
		require.NoError(t, err)
		assert.Equal(t, []string{"/usr/bin/codesign", "--verify", "--strict", "--deep", app}, f.calls[0])
	})
}
