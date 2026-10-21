package macpkg

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	shelllib "github.com/discentem/starcm/libraries/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

// tailscaleSigner and tailscaleSHA256 are the installer certificate in
// libraries/macsig/testdata/developer-id-notarized.txt, real
// `pkgutil --check-signature` output for a Developer ID signed, notarized
// package.
// pkgBytesSHA256 is the sha256 of "pkg bytes", the content writePkg uses.
const pkgBytesSHA256 = "fed2df1114f653c1184c3484ef40aea482f9634f49189b7ea5be36c204b8a1ca"

const (
	tailscaleSigner = "Developer ID Installer: Tailscale Inc. (W5364U7YZB)"
	tailscaleSHA256 = "1e77de6fe47d426984c5aa180d581f3246197f73c0c713277b844008b936e093"
)

func readTestdata(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "libraries", "macsig", "testdata", name))
	require.NoError(t, err)
	return string(b)
}

// fakeMac is a shelllib.Executor that records commands and answers pkgutil
// and installer the way macOS does for the outputs it's given.
type fakeMac struct {
	calls        [][]string
	pending      []string
	checkOutput  string
	checkErr     error
	pkgInfo      string // "" means not installed
	installerErr error
	// seen is the content of the package passed to pkgutil --check-signature.
	seen string
}

var _ shelllib.Executor = (*fakeMac)(nil)

func (f *fakeMac) Command(name string, args ...string) {
	f.pending = append([]string{name}, args...)
}

func (f *fakeMac) Stream(...io.WriteCloser) error { return errors.New("not used") }
func (f *fakeMac) ExitCode() (int, error)         { return 0, errors.New("not used") }

func (f *fakeMac) CombinedOutput() ([]byte, error) {
	name, args := f.pending[0], f.pending[1:]
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
		a := &pkgAction{executor: f, goos: "darwin"}
		pkg := writePkg(t, "pkg bytes")
		r, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String(pkg)}, kw("signer", tailscaleSigner))
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
		a := &pkgAction{executor: f, goos: "darwin"}
		_, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String(writePkg(t, "x"))}, kw("signer", "Developer ID Installer: Some Other Co (ABCDE12345)"))
		assert.ErrorContains(t, err, `signed by "`+tailscaleSigner+`", not the expected "Developer ID Installer: Some Other Co (ABCDE12345)"`)
		assert.Equal(t, []string{"pkgutil --check-signature"}, f.commands(), "installer never runs")
	})

	t.Run("refuses an unsigned package", func(t *testing.T) {
		f := &fakeMac{checkOutput: readTestdata(t, "unsigned.txt"), checkErr: errors.New("exit status 1")}
		a := &pkgAction{executor: f, goos: "darwin"}
		_, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String(writePkg(t, "x"))}, kw("signer", tailscaleSigner))
		assert.ErrorContains(t, err, "not signed with a certificate Apple issued for distribution: no signature")
		assert.Equal(t, []string{"pkgutil --check-signature"}, f.commands())
	})

	t.Run("what_if checks but doesn't install", func(t *testing.T) {
		f := &fakeMac{checkOutput: signed}
		a := &pkgAction{executor: f, goos: "darwin"}
		r, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String(writePkg(t, "x"))}, kw("signer", tailscaleSigner, "what_if", true))
		require.NoError(t, err)
		assert.True(t, r.Changed)
		assert.Contains(t, *r.Message, "would install")
		assert.Equal(t, []string{"pkgutil --check-signature"}, f.commands())
	})

	t.Run("skips an installed version", func(t *testing.T) {
		f := &fakeMac{checkOutput: signed, pkgInfo: "package-id: com.example.agent\nversion: 1.2.0\nvolume: /\n"}
		a := &pkgAction{executor: f, goos: "darwin"}
		r, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String(writePkg(t, "x"))}, kw("signer", tailscaleSigner, "id", "com.example.agent", "version", "1.2.0"))
		require.NoError(t, err)
		assert.False(t, r.Changed)
		assert.Equal(t, "com.example.agent 1.2.0 is already installed", *r.Message)
		assert.Equal(t, []string{"pkgutil --pkg-info"}, f.commands())
	})

	t.Run("installs when another version is installed", func(t *testing.T) {
		f := &fakeMac{checkOutput: signed, pkgInfo: "package-id: com.example.agent\nversion: 1.1.0\n"}
		a := &pkgAction{executor: f, goos: "darwin"}
		r, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String(writePkg(t, "x"))}, kw("signer", tailscaleSigner, "id", "com.example.agent", "version", "1.2.0", "target", "/Volumes/Other"))
		require.NoError(t, err)
		assert.True(t, r.Changed)
		assert.Equal(t, []string{"pkgutil --pkg-info", "pkgutil --check-signature", "installer -pkg"}, f.commands())
		assert.Equal(t, []string{"--volume", "/Volumes/Other"}, f.calls[0][3:])
		assert.Equal(t, []string{"-target", "/Volumes/Other"}, f.calls[2][3:])
	})

	t.Run("installer failure stops the run", func(t *testing.T) {
		f := &fakeMac{checkOutput: signed, installerErr: errors.New("exit status 1")}
		a := &pkgAction{executor: f, goos: "darwin"}
		_, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String(writePkg(t, "x"))}, kw("signer", tailscaleSigner))
		assert.ErrorContains(t, err, "installer failed: exit status 1")
	})

	t.Run("checks a pinned sha256 of the package", func(t *testing.T) {
		sum := strings.Repeat("ab", 32)
		f := &fakeMac{checkOutput: signed}
		a := &pkgAction{executor: f, goos: "darwin"}
		pkg := writePkg(t, "pkg bytes")
		_, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String(pkg)}, kw("signer", tailscaleSigner, "sha256", sum))
		assert.ErrorContains(t, err, "has sha256 ")
		assert.ErrorContains(t, err, ", not the expected "+sum)
		assert.Empty(t, f.calls, "pkgutil and installer never run")

		f = &fakeMac{checkOutput: signed}
		a = &pkgAction{executor: f, goos: "darwin"}
		r, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String(pkg)}, kw("signer", tailscaleSigner, "sha256", strings.ToUpper(pkgBytesSHA256)))
		require.NoError(t, err)
		assert.True(t, r.Changed)

		_, err = a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String(pkg)}, kw("signer", tailscaleSigner, "sha256", "abc"))
		assert.ErrorContains(t, err, `sha256 "abc" is not 64 hex characters`)
	})

	t.Run("re-checks files from the local store", func(t *testing.T) {
		f := &fakeMac{checkOutput: signed}
		a := &pkgAction{executor: f, goos: "darwin", checkStored: func(string) error { return errors.New("changed after it was stored") }}
		_, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String(writePkg(t, "x"))}, kw("signer", tailscaleSigner))
		assert.ErrorContains(t, err, "changed after it was stored")
		assert.Empty(t, f.calls)
	})

	t.Run("argument errors come before the platform check", func(t *testing.T) {
		a := &pkgAction{executor: &fakeMac{}, goos: "linux"}
		_, err := a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String("x.pkg")}, kw("signer", "  "))
		assert.ErrorContains(t, err, "signer must name the certificate")
		_, err = a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String("x.pkg")}, kw("signer", tailscaleSigner, "id", "com.example.agent"))
		assert.ErrorContains(t, err, "id and version must be given together")
		_, err = a.Run(ctx, "", "agent", nil, starlark.Tuple{starlark.String("x.pkg")}, kw("signer", tailscaleSigner))
		assert.ErrorContains(t, err, "install_mac_pkg() only works on macOS")
	})
}
