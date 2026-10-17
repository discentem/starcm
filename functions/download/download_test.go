package download

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/discentem/starcm/functions/base"
	localstore "github.com/discentem/starcm/libraries/local_store"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		action         *downloadAction
		moduleName     string
		starlarkArgs   starlark.Tuple
		starlarkKwargs []starlark.Tuple
		expectedResult func(*base.Result) bool
		expectedError  func(err error) bool
	}{
		{
			name: "Test download with nil fsys",
			action: &downloadAction{
				httpClient: &http.Client{},
				fsys:       nil,
			},
			moduleName: fmt.Sprintf("download %q", "hello world"),
			expectedResult: func(result *base.Result) bool {
				return result == nil
			},
			expectedError: func(err error) bool {
				return err != nil
			},
		},
		{
			name: "Test downloading succeeds",
			action: &downloadAction{
				httpClient: func() *http.Client {
					client := &http.Client{}
					client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(bytes.NewBufferString("hello world")),
						}, nil
					})
					return client
				}(),
				fsys: afero.NewMemMapFs(),
			},
			moduleName:   fmt.Sprintf("download %q", "hello world"),
			starlarkArgs: nil,
			starlarkKwargs: []starlark.Tuple{
				{
					starlark.String("url"),
					starlark.String("http://example.com"),
				},
				{
					starlark.String("save_to"),
					starlark.String("file.txt"),
				},
				{
					starlark.String("sha256"),
					starlark.String("b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"),
				},
			},
			expectedResult: func(result *base.Result) bool {
				return result != nil
			},
			expectedError: func(err error) bool {
				return err == nil
			},
		},
		{
			name: "Test download with missing url",
			action: &downloadAction{
				httpClient: http.DefaultClient,
				fsys:       afero.NewMemMapFs(),
			},
			moduleName:   fmt.Sprintf("download %q", "hello world"),
			starlarkArgs: nil,
			starlarkKwargs: []starlark.Tuple{
				{
					starlark.String("save_to"),
					starlark.String("file.txt"),
				},
			},
			expectedResult: func(result *base.Result) bool {
				return result == nil
			},
			expectedError: func(err error) bool {
				return err != nil
			},
		},
		{
			name: "Test download with missing save_to",
			action: &downloadAction{
				httpClient: http.DefaultClient,
				fsys:       afero.NewMemMapFs(),
			},
			moduleName:   fmt.Sprintf("download %q", "hello world"),
			starlarkArgs: nil,
			starlarkKwargs: []starlark.Tuple{
				{
					starlark.String("url"),
					starlark.String("http://example.com"),
				},
			},
			expectedResult: func(result *base.Result) bool {
				return result == nil
			},
			expectedError: func(err error) bool {
				return err != nil
			},
		},
		{
			name: "Test non-200 status code",
			action: &downloadAction{
				httpClient: func() *http.Client {
					client := &http.Client{}
					client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusNotFound,
						}, nil
					})
					return client
				}(),
				fsys: afero.NewMemMapFs(),
			},
			moduleName:   fmt.Sprintf("download %q", "hello world"),
			starlarkArgs: nil,
			starlarkKwargs: []starlark.Tuple{
				{
					starlark.String("url"),
					starlark.String("http://example.com"),
				},
				{
					starlark.String("save_to"),
					starlark.String("file.txt"),
				},
			},
			expectedResult: func(result *base.Result) bool {
				return result == nil
			},
			expectedError: func(err error) bool {
				return err != nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Running test %q", tt.name)
			thread := starlark.Thread{Name: "test"}
			result, err := tt.action.Run(context.TODO(), "", tt.moduleName, &thread, tt.starlarkArgs, tt.starlarkKwargs)
			if tt.expectedError == nil {
				t.Fatal("tt.expectedError must be provided")
			}
			if tt.expectedResult == nil {
				t.Fatal("tt.expectedResult must be provided")
			}
			t.Logf("result: %v", result)
			t.Logf("err: %v", err)

			assert.Equalf(t, true, tt.expectedError(err),
				"unexpected error: expectedError: %v, err: %v", tt.expectedError(err), err)
			assert.Equalf(t, true, tt.expectedResult(result),
				"unexpected result: expectedResult: %v, result: %v", tt.expectedResult(result), result)

			if result != nil {
				if result.Success == true {
					t.Logf("result.Output: %q", *result.Message)
					assert.Equal(t, "downloaded file to file.txt", *result.Message)
					f, err := tt.action.fsys.Open("file.txt")
					assert.NoError(t, err)
					b, err := io.ReadAll(f)
					assert.NoError(t, err)
					t.Logf("file.txt content: %q", string(b))
					assert.Equal(t, "hello world", string(b))
				}
			}
		})
	}
}

func serving(body string) *http.Client {
	return &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
		}, nil
	})}
}

func kwargs(pairs ...string) []starlark.Tuple {
	var out []starlark.Tuple
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, starlark.Tuple{starlark.String(pairs[i]), starlark.String(pairs[i+1])})
	}
	return out
}

func TestSaveToRenamesIntoPlace(t *testing.T) {
	fsys := afero.NewMemMapFs()
	a := &downloadAction{httpClient: serving("hello world"), fsys: fsys}

	hash := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	r, err := a.Run(context.Background(), "", "get", nil, nil,
		kwargs("url", "http://example.com/hello.txt", "save_to", "out/hello.txt", "sha256", hash))
	require.NoError(t, err)
	require.True(t, r.Changed)

	fi, err := fsys.Stat("out/hello.txt")
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), fi.Mode().Perm())

	entries, err := afero.ReadDir(fsys, "out")
	require.NoError(t, err)
	require.Len(t, entries, 1, "no temporary file may be left behind")

	_, err = a.Run(context.Background(), "", "bad", nil, nil,
		kwargs("url", "http://example.com/hello.txt", "save_to", "out/other.txt", "sha256", strings.Repeat("0", 64)))
	require.ErrorContains(t, err, "expected sha256 hash")
	entries, err = afero.ReadDir(fsys, "out")
	require.NoError(t, err)
	require.Len(t, entries, 1, "a failed download must leave nothing behind")
}

func TestStoreOnlyRefusesSaveTo(t *testing.T) {
	a := &downloadAction{httpClient: serving("hello world"), fsys: afero.NewMemMapFs(), storeOnly: true}
	_, err := a.Run(context.Background(), "", "get", nil, nil,
		kwargs("url", "http://example.com/hello.txt", "save_to", "hello.txt", "sha256", strings.Repeat("0", 64)))
	require.ErrorContains(t, err, "save_to is disabled by --store-only")
}

func TestWithoutSaveToUsesStore(t *testing.T) {
	perms, err := localstore.NewPermissions()
	require.NoError(t, err)
	a := &downloadAction{
		httpClient: serving("hello world"),
		fsys:       afero.NewMemMapFs(),
		store:      localstore.Lazy(filepath.Join(t.TempDir(), "store"), perms),
	}
	hash := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

	r, err := a.Run(context.Background(), "", "get", nil, nil,
		kwargs("url", "http://example.com/dl/hello.txt?x=1", "sha256", hash))
	require.NoError(t, err)
	require.True(t, r.Changed)
	path := string(r.Return.(starlark.String))
	require.True(t, strings.HasSuffix(path, filepath.Join("sha256", hash, "hello.txt")), path)

	r, err = a.Run(context.Background(), "", "again", nil, nil,
		kwargs("url", "http://example.com/dl/hello.txt", "sha256", hash))
	require.NoError(t, err)
	require.False(t, r.Changed)

	_, err = a.Run(context.Background(), "", "unnamed", nil, nil,
		kwargs("url", "http://example.com/", "sha256", hash))
	require.ErrorContains(t, err, "pass file_name")
}

func TestWithoutSaveToOrStore(t *testing.T) {
	a := &downloadAction{httpClient: serving("hello world"), fsys: afero.NewMemMapFs()}
	_, err := a.Run(context.Background(), "", "get", nil, nil,
		kwargs("url", "http://example.com/hello.txt", "sha256", strings.Repeat("0", 64)))
	require.ErrorContains(t, err, "no store is configured")
}
