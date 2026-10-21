package checkmacsig

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	shelllib "github.com/discentem/starcm/libraries/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

const tailscaleSigner = "Developer ID Installer: Tailscale Inc. (W5364U7YZB)"

// fakePkgutil answers `pkgutil --check-signature` with output, and records
// the content of the file it was asked about.
type fakePkgutil struct {
	pending []string
	output  string
	seen    string
}

var _ shelllib.Executor = (*fakePkgutil)(nil)

func (f *fakePkgutil) Command(name string, args ...string) {
	f.pending = append([]string{name}, args...)
}
func (f *fakePkgutil) Stream(...io.WriteCloser) error { return errors.New("not used") }
func (f *fakePkgutil) ExitCode() (int, error)         { return 0, errors.New("not used") }

func (f *fakePkgutil) CombinedOutput() ([]byte, error) {
	if f.pending[0] != "/usr/sbin/pkgutil" || f.pending[1] != "--check-signature" {
		return nil, errors.New("unexpected command")
	}
	b, err := os.ReadFile(f.pending[2])
	if err != nil {
		return nil, err
	}
	f.seen = string(b)
	return []byte(f.output), nil
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

func TestRun(t *testing.T) {
	ctx := context.Background()
	out, err := os.ReadFile(filepath.Join("..", "..", "libraries", "macsig", "testdata", "developer-id-notarized.txt"))
	require.NoError(t, err)
	pkg := filepath.Join(t.TempDir(), "agent.pkg")
	require.NoError(t, os.WriteFile(pkg, []byte("pkg bytes"), 0o644))
	const pkgSHA256 = "fed2df1114f653c1184c3484ef40aea482f9634f49189b7ea5be36c204b8a1ca"

	t.Run("reports the signature and sha256 of what it checked", func(t *testing.T) {
		f := &fakePkgutil{output: string(out)}
		a := &checkAction{executor: f, goos: "darwin"}
		r, err := a.Run(ctx, "", "check", nil, starlark.Tuple{starlark.String(pkg)}, kw("signer", tailscaleSigner))
		require.NoError(t, err)
		assert.True(t, r.Success)
		assert.False(t, r.Changed)
		assert.Equal(t, "pkg bytes", f.seen)
		assert.Equal(t, pkg+` is signed by "`+tailscaleSigner+`"`, *r.Message)
		v := r.Return.(*starlarkstruct.Struct)
		for field, want := range map[string]starlark.Value{
			"signer":    starlark.String(tailscaleSigner),
			"team_id":   starlark.String("W5364U7YZB"),
			"notarized": starlark.True,
			"sha256":    starlark.String(pkgSHA256),
			"kind":      starlark.String("pkg"),
		} {
			got, err := v.Attr(field)
			require.NoError(t, err)
			assert.Equal(t, want, got, field)
		}
	})

	t.Run("another signer stops the run", func(t *testing.T) {
		a := &checkAction{executor: &fakePkgutil{output: string(out)}, goos: "darwin"}
		_, err := a.Run(ctx, "", "check", nil, starlark.Tuple{starlark.String(pkg)}, kw("signer", "Developer ID Installer: Example Corp (ABCDE12345)"))
		assert.EqualError(t, err, `check_mac_signature(label="check"): `+pkg+`: signed by "`+tailscaleSigner+`", not the expected "Developer ID Installer: Example Corp (ABCDE12345)"`)
	})

	t.Run("store files are re-hashed first", func(t *testing.T) {
		a := &checkAction{executor: &fakePkgutil{output: string(out)}, goos: "darwin", checkStored: func(p string) error {
			return errors.New("tampered: " + p)
		}}
		_, err := a.Run(ctx, "", "check", nil, starlark.Tuple{starlark.String(pkg)}, kw("signer", tailscaleSigner))
		assert.EqualError(t, err, `check_mac_signature(label="check"): tampered: `+pkg)
	})

	t.Run("argument errors", func(t *testing.T) {
		a := &checkAction{executor: &fakePkgutil{}, goos: "darwin"}
		_, err := a.Run(ctx, "", "check", nil, starlark.Tuple{starlark.String(pkg)}, kw("signer", " "))
		assert.ErrorContains(t, err, "signer must name the certificate")
		_, err = a.Run(ctx, "", "check", nil, starlark.Tuple{starlark.String(pkg)}, kw("signer", tailscaleSigner, "kind", "dmg"))
		assert.EqualError(t, err, `check_mac_signature(label="check"): kind must be "auto", "pkg" or "code", not "dmg"`)
		linux := &checkAction{executor: &fakePkgutil{}, goos: "linux"}
		_, err = linux.Run(ctx, "", "check", nil, starlark.Tuple{starlark.String(pkg)}, kw("signer", tailscaleSigner))
		assert.EqualError(t, err, `check_mac_signature(label="check"): check_mac_signature() only works on macOS`)
	})
}
