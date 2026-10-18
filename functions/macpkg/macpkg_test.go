package macpkg

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

// tailscaleSHA256 is the installer certificate in
// testdata/developer-id-notarized.txt, real `pkgutil --check-signature`
// output for a Developer ID signed, notarized package.
const tailscaleSHA256 = "1e77de6fe47d426984c5aa180d581f3246197f73c0c713277b844008b936e093"

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
	assert.NoError(t, sig.Check(tailscaleSHA256, true))

	other := strings.Repeat("ab", 32)
	assert.ErrorContains(t, sig.Check(other, true), `signed by "Developer ID Installer: Tailscale Inc. (W5364U7YZB)" with certificate sha256 `+tailscaleSHA256+", not the expected "+other)

	// Pinning the root or intermediate isn't enough: only the leaf counts.
	assert.Error(t, sig.Check(sig.Chain[2].SHA256, true))

	notNotarized := *sig
	notNotarized.Notarized = false
	assert.ErrorContains(t, notNotarized.Check(tailscaleSHA256, true), "not notarized")
	assert.NoError(t, notNotarized.Check(tailscaleSHA256, false))

	for _, status := range []string{"no signature", "signed by untrusted certificate", "signed by a certificate that has since expired", "signed by a developer certificate issued by Apple (Development)"} {
		s := *sig
		s.Status = status
		assert.ErrorContains(t, s.Check(tailscaleSHA256, false), "signature is not trusted: "+status)
	}
}

func TestNormalizeFingerprint(t *testing.T) {
	for _, in := range []string{
		"1E 77 DE 6F E4 7D 42 69 84 C5 AA 18 0D 58 1F 32 46 19 7F 73 C0 C7 \n 13 27 7B 84 40 08 B9 36 E0 93",
		"1E:77:DE:6F:E4:7D:42:69:84:C5:AA:18:0D:58:1F:32:46:19:7F:73:C0:C7:13:27:7B:84:40:08:B9:36:E0:93",
		tailscaleSHA256,
	} {
		got, err := NormalizeFingerprint(in)
		require.NoError(t, err, in)
		assert.Equal(t, tailscaleSHA256, got)
	}
	for _, in := range []string{"", "W5364U7YZB", tailscaleSHA256[:62], tailscaleSHA256 + "00", strings.Repeat("zz", 32)} {
		_, err := NormalizeFingerprint(in)
		assert.Error(t, err, in)
	}
}

// fakeMac records commands and answers pkgutil and installer the way macOS
// does for the outputs it's given.
type fakeMac struct {
	calls        [][]string
	checkOutput  string
	checkErr     error
	pkgInfo      string // "" means not installed
	installerErr error
	// seen is the content of the package passed to pkgutil --check-signature.
	seen string
}

func (f *fakeMac) run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{filepath.Base(name)}, args...))
	switch {
	case filepath.Base(name) == "pkgutil" && args[0] == "--check-signature":
		b, err := os.ReadFile(args[1])
		if err != nil {
			return nil, err
		}
		f.seen = string(b)
		return []byte(f.checkOutput), f.checkErr
	case filepath.Base(name) == "pkgutil" && args[0] == "--pkg-info":
		if f.pkgInfo == "" {
			return []byte("No receipt for '" + args[1] + "' found at '/'.\n"), errors.New("exit status 1")
		}
		return []byte(f.pkgInfo), nil
	case filepath.Base(name) == "installer":
		if f.installerErr != nil {
			return []byte("installer: Error - the package path specified was invalid"), f.installerErr
		}
		return []byte("installer: The install was successful.\n"), nil
	}
	return nil, errors.New("unexpected command " + name)
}

func (f *fakeMac) commands() []string {
	var out []string
	for _, c := range f.calls {
		out = append(out, c[0]+" "+c[1])
	}
	return out
}

func kw(pairs ...any) []starlark.Tuple {
	var out []starlark.Tuple
	for i := 0; i < len(pairs); i += 2 {
		var v starlark.Value
		switch x := pairs[i+1].(type) {
		case string:
			v = starlark.String(x)
		case bool:
			v = starlark.Bool(x)
		}
		out = append(out, starlark.Tuple{starlark.String(pairs[i].(string)), v})
	}
	return out
}

func writePkg(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "agent.pkg")
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func TestRun(t *testing.T) {
	ctx := context.Background()
	signed := readTestdata(t, "developer-id-notarized.txt")

	t.Run("installs a correctly signed package", func(t *testing.T) {
		f := &fakeMac{checkOutput: signed}
		a := &pkgAction{run: f.run, goos: "darwin"}
		pkg := writePkg(t, "pkg bytes")
		r, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String(pkg)}, kw("signer_sha256", strings.ToUpper(tailscaleSHA256)))
		require.NoError(t, err)
		assert.True(t, r.Success)
		assert.True(t, r.Changed)
		assert.Equal(t, `installed `+pkg+`, signed by "Developer ID Installer: Tailscale Inc. (W5364U7YZB)"`, *r.Message)
		assert.Equal(t, []string{"pkgutil --check-signature", "installer -pkg"}, f.commands())
		// The copy that was checked is the one installed, and it isn't the original path.
		assert.Equal(t, "pkg bytes", f.seen)
		checked, installed := f.calls[0][2], f.calls[1][2]
		assert.Equal(t, checked, installed)
		assert.NotEqual(t, pkg, checked)
		assert.Equal(t, []string{"-target", "/"}, f.calls[1][3:])
		_, err = os.Stat(checked)
		assert.True(t, os.IsNotExist(err), "temporary copy is removed")

		v := r.Return.(interface {
			Attr(string) (starlark.Value, error)
		})
		team, _ := v.Attr("team_id")
		assert.Equal(t, starlark.String("W5364U7YZB"), team)
	})

	t.Run("refuses a package signed by someone else", func(t *testing.T) {
		f := &fakeMac{checkOutput: signed}
		a := &pkgAction{run: f.run, goos: "darwin"}
		_, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String(writePkg(t, "x"))}, kw("signer_sha256", strings.Repeat("00", 32)))
		assert.ErrorContains(t, err, "not the expected 0000")
		assert.Equal(t, []string{"pkgutil --check-signature"}, f.commands(), "installer never runs")
	})

	t.Run("refuses an unsigned package", func(t *testing.T) {
		f := &fakeMac{checkOutput: readTestdata(t, "unsigned.txt"), checkErr: errors.New("exit status 1")}
		a := &pkgAction{run: f.run, goos: "darwin"}
		_, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String(writePkg(t, "x"))}, kw("signer_sha256", tailscaleSHA256))
		assert.ErrorContains(t, err, "signature is not trusted: no signature")
		assert.Equal(t, []string{"pkgutil --check-signature"}, f.commands())
	})

	t.Run("what_if checks but doesn't install", func(t *testing.T) {
		f := &fakeMac{checkOutput: signed}
		a := &pkgAction{run: f.run, goos: "darwin"}
		r, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String(writePkg(t, "x"))}, kw("signer_sha256", tailscaleSHA256, "what_if", true))
		require.NoError(t, err)
		assert.True(t, r.Changed)
		assert.Contains(t, *r.Message, "would install")
		assert.Equal(t, []string{"pkgutil --check-signature"}, f.commands())
	})

	t.Run("skips an installed version", func(t *testing.T) {
		f := &fakeMac{checkOutput: signed, pkgInfo: "package-id: com.example.agent\nversion: 1.2.0\nvolume: /\n"}
		a := &pkgAction{run: f.run, goos: "darwin"}
		r, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String(writePkg(t, "x"))}, kw("signer_sha256", tailscaleSHA256, "id", "com.example.agent", "version", "1.2.0"))
		require.NoError(t, err)
		assert.False(t, r.Changed)
		assert.Equal(t, "com.example.agent 1.2.0 is already installed", *r.Message)
		assert.Equal(t, []string{"pkgutil --pkg-info"}, f.commands())
	})

	t.Run("installs when another version is installed", func(t *testing.T) {
		f := &fakeMac{checkOutput: signed, pkgInfo: "package-id: com.example.agent\nversion: 1.1.0\n"}
		a := &pkgAction{run: f.run, goos: "darwin"}
		r, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String(writePkg(t, "x"))}, kw("signer_sha256", tailscaleSHA256, "id", "com.example.agent", "version", "1.2.0", "target", "/Volumes/Other"))
		require.NoError(t, err)
		assert.True(t, r.Changed)
		assert.Equal(t, []string{"pkgutil --pkg-info", "pkgutil --check-signature", "installer -pkg"}, f.commands())
		assert.Equal(t, []string{"--volume", "/Volumes/Other"}, f.calls[0][3:])
		assert.Equal(t, []string{"-target", "/Volumes/Other"}, f.calls[2][3:])
	})

	t.Run("installer failure stops the run", func(t *testing.T) {
		f := &fakeMac{checkOutput: signed, installerErr: errors.New("exit status 1")}
		a := &pkgAction{run: f.run, goos: "darwin"}
		_, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String(writePkg(t, "x"))}, kw("signer_sha256", tailscaleSHA256))
		assert.ErrorContains(t, err, "installer failed: exit status 1")
	})

	t.Run("re-checks files from the local store", func(t *testing.T) {
		f := &fakeMac{checkOutput: signed}
		a := &pkgAction{run: f.run, goos: "darwin", checkStored: func(string) error { return errors.New("changed after it was stored") }}
		_, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String(writePkg(t, "x"))}, kw("signer_sha256", tailscaleSHA256))
		assert.ErrorContains(t, err, "changed after it was stored")
		assert.Empty(t, f.calls)
	})

	t.Run("argument errors come before the platform check", func(t *testing.T) {
		a := &pkgAction{run: (&fakeMac{}).run, goos: "linux"}
		_, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String("x.pkg")}, kw("signer_sha256", "W5364U7YZB"))
		assert.ErrorContains(t, err, "signer_sha256")
		_, err = a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String("x.pkg")}, kw("signer_sha256", tailscaleSHA256, "id", "com.example.agent"))
		assert.ErrorContains(t, err, "id and version must be given together")
		_, err = a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String("x.pkg")}, kw("signer_sha256", tailscaleSHA256))
		assert.ErrorContains(t, err, "install_mac_pkg() only works on macOS")
	})
}
