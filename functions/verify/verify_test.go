package verify

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/discentem/starcm/libraries/signing"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// setup returns a directory with a keyring trusting key "test", and a
// function that writes a file signed with that key (or unsigned).
func setup(t *testing.T) (dir string, keyring *signing.Keyring, write func(name, content string, signed bool) string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)
	dir = t.TempDir()
	keys := filepath.Join(dir, "trusted")
	require.NoError(t, os.Mkdir(keys, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(keys, "test.pub.pem"), pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), 0o644))
	keyring, err = signing.LoadKeyring(afero.NewOsFs(), keys)
	require.NoError(t, err)
	write = func(name, content string, signed bool) string {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
		if signed {
			sig, err := signing.Sign(priv, "test", "2026.10", time.Now().Add(time.Hour).UTC().Truncate(time.Second), []byte(content))
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(p+signing.SignatureSuffix, sig.Bytes(), 0o644))
		}
		return p
	}
	return dir, keyring, write
}

func call(t *testing.T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	t.Helper()
	return starlark.Call(&starlark.Thread{}, b, starlark.Tuple(args), nil)
}

func field(t *testing.T, v starlark.Value, name string) starlark.Value {
	t.Helper()
	s, ok := v.(*starlarkstruct.Struct)
	require.True(t, ok, "got %s, want struct", v.Type())
	f, err := s.Attr(name)
	require.NoError(t, err)
	return f
}

func TestVerifyAll(t *testing.T) {
	_, keyring, write := setup(t)
	verifyAll := NewAll(context.Background(), afero.NewOsFs(), keyring).Builtin()
	a := write("a.txt", "a", true)
	b := write("b.txt", "b", true)

	v, err := call(t, verifyAll, starlark.NewList([]starlark.Value{starlark.String(a), starlark.String(b)}))
	require.NoError(t, err)
	list := v.(*starlark.List)
	require.Equal(t, 2, list.Len())
	assert.Equal(t, starlark.String(a), field(t, list.Index(0), "path"))
	assert.Equal(t, starlark.String(b), field(t, list.Index(1), "path"))
	assert.Equal(t, starlark.String("test"), field(t, list.Index(0), "key_id"))
	assert.Equal(t, starlark.String("2026.10"), field(t, list.Index(0), "release"))

	// Nothing to check is fine.
	v, err = call(t, verifyAll, starlark.NewList(nil))
	require.NoError(t, err)
	assert.Equal(t, 0, v.(*starlark.List).Len())
}

// verify_all checks every file before stopping the run, and lists each one
// that failed, in order.
func TestVerifyAllReportsEveryFailure(t *testing.T) {
	_, keyring, write := setup(t)
	verifyAll := NewAll(context.Background(), afero.NewOsFs(), keyring).Builtin()
	good := write("good.txt", "good", true)
	tampered := write("tampered.txt", "payload", true)
	require.NoError(t, os.WriteFile(tampered, []byte("malware"), 0o644))
	unsigned := write("unsigned.txt", "payload", false)

	_, err := call(t, verifyAll, starlark.NewList([]starlark.Value{
		starlark.String(tampered), starlark.String(good), starlark.String(unsigned),
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "verify_all: 2 of 3 files didn't verify:\n  "+tampered+`: key "test": signature does not match`+"\n  "+unsigned+": reading signature: ")
	assert.NotContains(t, err.Error(), good)
}

func TestVerifyAllArgumentErrors(t *testing.T) {
	_, keyring, _ := setup(t)
	verifyAll := NewAll(context.Background(), afero.NewOsFs(), keyring).Builtin()
	_, err := call(t, verifyAll, starlark.NewList([]starlark.Value{starlark.MakeInt(1)}))
	assert.EqualError(t, err, "verify_all: paths[0]: got int, want string")
	_, err = call(t, verifyAll, starlark.NewList([]starlark.Value{starlark.String("rel.txt")}))
	assert.ErrorContains(t, err, `verify_all: paths[0]: "rel.txt" is not an absolute path`)
	_, err = call(t, verifyAll, starlark.String("/a"))
	assert.ErrorContains(t, err, "verify_all: for parameter paths: got string, want list")
}

func TestNoKeys(t *testing.T) {
	_, err := call(t, NewAll(context.Background(), afero.NewOsFs(), nil).Builtin(), starlark.NewList(nil))
	assert.EqualError(t, err, "verify_all: needs trusted keys: run starcm with --trusted-keys")
}
