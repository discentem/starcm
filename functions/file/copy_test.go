package file

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	shelllib "github.com/discentem/starcm/libraries/shell"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func kw(pairs ...any) []starlark.Tuple {
	var out []starlark.Tuple
	for i := 0; i < len(pairs); i += 2 {
		var v starlark.Value
		switch x := pairs[i+1].(type) {
		case string:
			v = starlark.String(x)
		case bool:
			v = starlark.Bool(x)
		case int:
			v = starlark.MakeInt(x)
		case starlark.Value:
			v = x
		}
		out = append(out, starlark.Tuple{starlark.String(pairs[i].(string)), v})
	}
	return out
}

// fakePkgutil answers `pkgutil --check-signature` with output, and records
// which file it was asked about and what it held.
type fakePkgutil struct {
	pending  []string
	output   string
	path     string
	content  string
	onVerify func()
}

var _ shelllib.Executor = (*fakePkgutil)(nil)

func (f *fakePkgutil) Command(name string, args ...string) {
	f.pending = append([]string{name}, args...)
}
func (f *fakePkgutil) Stream(...io.WriteCloser) error { return errors.New("not used") }
func (f *fakePkgutil) ExitCode() (int, error)         { return 0, errors.New("not used") }

func (f *fakePkgutil) CombinedOutput() ([]byte, error) {
	if f.pending[0] != "/usr/sbin/pkgutil" {
		return nil, errors.New("unexpected command " + f.pending[0])
	}
	f.path = f.pending[2]
	b, err := os.ReadFile(f.path)
	if err != nil {
		return nil, err
	}
	f.content = string(b)
	if f.onVerify != nil {
		f.onVerify()
	}
	return []byte(f.output), nil
}

const tailscaleSigner = "Developer ID Installer: Tailscale Inc. (W5364U7YZB)"

func TestCopy(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	src := filepath.Join(dir, "agent.pkg")
	require.NoError(t, os.WriteFile(src, []byte("pkg bytes"), 0o644))
	const srcSHA256 = "fed2df1114f653c1184c3484ef40aea482f9634f49189b7ea5be36c204b8a1ca"
	signed, err := os.ReadFile(filepath.Join("..", "..", "libraries", "macsig", "testdata", "developer-id-notarized.txt"))
	require.NoError(t, err)

	run := func(a *fileAction, args ...any) (string, bool, error) {
		r, err := a.Run(ctx, "", "copy", nil, nil, kw(args...))
		if err != nil {
			return "", false, err
		}
		return *r.Message, r.Changed, nil
	}

	t.Run("copies once, then does nothing", func(t *testing.T) {
		a := &fileAction{fsys: afero.NewOsFs(), goos: runtime.GOOS}
		dst := filepath.Join(dir, "out", "a.pkg")
		msg, changed, err := run(a, "path", dst, "source", src, "create_dirs", true)
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, fmt.Sprintf("copied %q to %q", src, dst), msg)
		b, err := os.ReadFile(dst)
		require.NoError(t, err)
		assert.Equal(t, "pkg bytes", string(b))

		msg, changed, err = run(a, "path", dst, "source", src, "sha256", srcSHA256)
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, fmt.Sprintf("file %q already matches %q", dst, src), msg)

		// Different content is replaced.
		require.NoError(t, os.WriteFile(dst, []byte("old"), 0o644))
		_, changed, err = run(a, "path", dst, "source", src)
		require.NoError(t, err)
		assert.True(t, changed)

		// No staging directories are left behind.
		entries, err := os.ReadDir(filepath.Dir(dst))
		require.NoError(t, err)
		assert.Len(t, entries, 1)
	})

	t.Run("mode", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows only keeps a read-only bit")
		}
		a := &fileAction{fsys: afero.NewOsFs(), goos: runtime.GOOS}
		dst := filepath.Join(dir, "mode.pkg")
		_, _, err := run(a, "path", dst, "source", src)
		require.NoError(t, err)
		info, err := os.Stat(dst)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o644), info.Mode().Perm(), "the source's mode by default")

		_, changed, err := run(a, "path", dst, "source", src, "mode", 0o755)
		require.NoError(t, err)
		assert.True(t, changed, "a different mode is a change")
		info, err = os.Stat(dst)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())

		_, changed, err = run(a, "path", dst, "source", src, "mode", 0o755)
		require.NoError(t, err)
		assert.False(t, changed)
	})

	t.Run("refusals leave the destination alone", func(t *testing.T) {
		dst := filepath.Join(dir, "kept.pkg")
		require.NoError(t, os.WriteFile(dst, []byte("kept"), 0o644))
		a := &fileAction{fsys: afero.NewOsFs(), goos: "darwin", executor: &fakePkgutil{output: string(signed)}}

		_, _, err := run(a, "path", dst, "source", src, "sha256", strings.Repeat("0", 64))
		assert.EqualError(t, err, `file(label="copy"): `+src+` has sha256 `+srcSHA256+`, not the expected `+strings.Repeat("0", 64))
		_, _, err = run(a, "path", dst, "source", src, "signer", "Developer ID Installer: Example Corp (ABCDE12345)")
		assert.EqualError(t, err, `file(label="copy"): `+src+`: signed by "`+tailscaleSigner+`", not the expected "Developer ID Installer: Example Corp (ABCDE12345)"`)
		a.checkStored = func(p string) error { return errors.New("tampered: " + p) }
		_, _, err = run(a, "path", dst, "source", src, "signer", tailscaleSigner)
		assert.EqualError(t, err, `file(label="copy"): tampered: `+src)

		b, err := os.ReadFile(dst)
		require.NoError(t, err)
		assert.Equal(t, "kept", string(b))
		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		for _, e := range entries {
			assert.False(t, strings.HasPrefix(e.Name(), ".starcm-file-"), e.Name())
		}
	})

	t.Run("the signature is checked on the copy that's installed", func(t *testing.T) {
		dst := filepath.Join(dir, "signed.pkg")
		f := &fakePkgutil{output: string(signed)}
		// Swapping the source after the copy was made changes nothing.
		f.onVerify = func() { require.NoError(t, os.WriteFile(src, []byte("evil"), 0o644)) }
		t.Cleanup(func() { os.WriteFile(src, []byte("pkg bytes"), 0o644) })
		a := &fileAction{fsys: afero.NewOsFs(), goos: "darwin", executor: f}
		r, err := a.Run(ctx, "", "copy", nil, nil, kw("path", dst, "source", src, "signer", tailscaleSigner))
		require.NoError(t, err)
		assert.NotEqual(t, src, f.path)
		assert.Equal(t, dir, filepath.Dir(filepath.Dir(f.path)), "staged next to the destination")
		assert.Equal(t, "pkg bytes", f.content)
		b, err := os.ReadFile(dst)
		require.NoError(t, err)
		assert.Equal(t, "pkg bytes", string(b))
		sum, err := r.Return.(starlark.HasAttrs).Attr("sha256")
		require.NoError(t, err)
		assert.Equal(t, starlark.String(srcSHA256), sum)
	})

	t.Run("argument errors", func(t *testing.T) {
		a := &fileAction{fsys: afero.NewOsFs(), goos: "linux"}
		dst := filepath.Join(dir, "x")
		_, _, err := run(a, "path", dst, "source", src, "content", "x")
		assert.EqualError(t, err, `file(label="copy"): give content or source, not both`)
		_, _, err = run(a, "path", dst, "source", src, "signer", tailscaleSigner)
		assert.EqualError(t, err, `file(label="copy"): signer only works on macOS`)
		_, _, err = run(a, "path", dst, "content", "x", "signer", tailscaleSigner)
		assert.EqualError(t, err, `file(label="copy"): signer needs source`)
		_, _, err = run(a, "path", dst, "source", src, "sha256", "abc")
		assert.EqualError(t, err, `file(label="copy"): sha256 "abc" is not 64 hex characters`)
		_, _, err = run(a, "path", dst, "source", dir)
		assert.EqualError(t, err, fmt.Sprintf(`file(label="copy"): source %q is not a regular file`, dir))
	})
}
