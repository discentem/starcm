package download

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
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
		action         *Fetcher
		moduleName     string
		starlarkKwargs []starlark.Tuple
		expectedResult func(*base.Result) bool
		expectedError  func(err error) bool
	}{
		{
			name: "Test download with nil fsys",
			action: &Fetcher{
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
			action: &Fetcher{
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
			moduleName: fmt.Sprintf("download %q", "hello world"),
			starlarkKwargs: []starlark.Tuple{
				{
					starlark.String("url"),
					starlark.String("http://example.com"),
				},
				{
					starlark.String("save_to"),
					starlark.String(abs("file.txt")),
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
			action: &Fetcher{
				httpClient: http.DefaultClient,
				fsys:       afero.NewMemMapFs(),
			},
			moduleName: fmt.Sprintf("download %q", "hello world"),
			starlarkKwargs: []starlark.Tuple{
				{
					starlark.String("save_to"),
					starlark.String(abs("file.txt")),
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
			action: &Fetcher{
				httpClient: http.DefaultClient,
				fsys:       afero.NewMemMapFs(),
			},
			moduleName: fmt.Sprintf("download %q", "hello world"),
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
			action: &Fetcher{
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
			moduleName: fmt.Sprintf("download %q", "hello world"),
			starlarkKwargs: []starlark.Tuple{
				{
					starlark.String("url"),
					starlark.String("http://example.com"),
				},
				{
					starlark.String("save_to"),
					starlark.String(abs("file.txt")),
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
			result, err := runURL(tt.action, tt.moduleName, tt.starlarkKwargs)
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
					assert.Equal(t, "downloaded file to "+abs("file.txt"), *result.Message)
					assert.Equal(t, starlark.String(abs("file.txt")), result.Return)
					f, err := tt.action.fsys.Open(abs("file.txt"))
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

// testRoot is an absolute directory for test paths on any OS.
func testRoot() string {
	if runtime.GOOS == "windows" {
		return `C:\repo`
	}
	return "/repo"
}

// abs is name under testRoot, since save_to must be absolute.
func abs(name string) string {
	return filepath.Join(testRoot(), filepath.FromSlash(name))
}

// request builds the Request download() and s3_download() would for these
// keyword arguments.
func request(fn, label string, kw []starlark.Tuple) (Request, map[string]string) {
	m := map[string]string{}
	for _, kv := range kw {
		m[string(kv[0].(starlark.String))] = string(kv[1].(starlark.String))
	}
	return Request{
		Function:  fn,
		Label:     label,
		SaveTo:    m["save_to"],
		FileName:  m["file_name"],
		SHA256:    m["sha256"],
		Signature: m["signature"],
	}, m
}

// runURL fetches as download() would for these keyword arguments and
// returns its result.
func runURL(f *Fetcher, label string, kw []starlark.Tuple) (*base.Result, error) {
	req, m := request("download", label, kw)
	got, err := f.FetchURL(context.Background(), m["url"], req)
	if err != nil {
		return nil, err
	}
	return resultFrom(label, got), nil
}

// runS3 is runURL for s3_download().
func runS3(f *Fetcher, label string, kw []starlark.Tuple) (*base.Result, error) {
	req, m := request("s3_download", label, kw)
	obj := S3Object{Bucket: m["bucket"], Key: m["key"], Endpoint: m["endpoint"], Region: m["region"]}
	got, err := f.FetchS3(context.Background(), obj, req)
	if err != nil {
		return nil, err
	}
	return resultFrom(label, got), nil
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
	a := &Fetcher{httpClient: serving("hello world"), fsys: fsys}

	hash := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	r, err := runURL(a, "get", kwargs("url", "http://example.com/hello.txt", "save_to", abs("out/hello.txt"), "sha256", hash))
	require.NoError(t, err)
	require.True(t, r.Changed)

	fi, err := fsys.Stat(abs("out/hello.txt"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), fi.Mode().Perm())

	entries, err := afero.ReadDir(fsys, abs("out"))
	require.NoError(t, err)
	require.Len(t, entries, 1, "no temporary file may be left behind")

	_, err = runURL(a, "bad", kwargs("url", "http://example.com/hello.txt", "save_to", abs("out/other.txt"), "sha256", strings.Repeat("0", 64)))
	require.ErrorContains(t, err, "expected sha256 hash")
	entries, err = afero.ReadDir(fsys, abs("out"))
	require.NoError(t, err)
	require.Len(t, entries, 1, "a failed download must leave nothing behind")
}

func TestStoreOnlyRefusesSaveTo(t *testing.T) {
	a := &Fetcher{httpClient: serving("hello world"), fsys: afero.NewMemMapFs(), storeOnly: true}
	_, err := runURL(a, "get", kwargs("url", "http://example.com/hello.txt", "save_to", abs("hello.txt"), "sha256", strings.Repeat("0", 64)))
	require.ErrorContains(t, err, "save_to is disabled by --store-only")
}

func TestWithoutSaveToUsesStore(t *testing.T) {
	perms, err := localstore.NewPermissions()
	require.NoError(t, err)
	a := &Fetcher{
		httpClient: serving("hello world"),
		fsys:       afero.NewMemMapFs(),
		store:      localstore.Lazy(filepath.Join(t.TempDir(), "store"), perms),
	}
	hash := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

	r, err := runURL(a, "get", kwargs("url", "http://example.com/dl/hello.txt?x=1", "sha256", hash))
	require.NoError(t, err)
	require.True(t, r.Changed)
	path := string(r.Return.(starlark.String))
	require.True(t, strings.HasSuffix(path, filepath.Join("sha256", hash, "hello.txt")), path)

	r, err = runURL(a, "again", kwargs("url", "http://example.com/dl/hello.txt", "sha256", hash))
	require.NoError(t, err)
	require.False(t, r.Changed)

	_, err = runURL(a, "unnamed", kwargs("url", "http://example.com", "sha256", hash))
	require.ErrorContains(t, err, "pass file_name")
}

func TestWithoutSaveToOrStore(t *testing.T) {
	a := &Fetcher{httpClient: serving("hello world"), fsys: afero.NewMemMapFs()}
	_, err := runURL(a, "get", kwargs("url", "http://example.com/hello.txt", "sha256", strings.Repeat("0", 64)))
	require.ErrorContains(t, err, "no store is configured")
}
