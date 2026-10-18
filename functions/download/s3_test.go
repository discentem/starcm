package download

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/discentem/starcm/functions/base"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

const helloSHA256 = "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9" // "hello world"

// fakeS3 serves objects at /<bucket>/<key> (path-style) and records each
// request's Authorization header.
type fakeS3 struct {
	mu      sync.Mutex
	objects map[string]string
	auth    []string
	paths   []string
}

func (f *fakeS3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.auth = append(f.auth, r.Header.Get("Authorization"))
	f.paths = append(f.paths, r.URL.Path)
	f.mu.Unlock()
	body, ok := f.objects[r.URL.Path]
	if !ok {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>NoSuchKey</Code><Message>The specified key does not exist.</Message></Error>`)
		return
	}
	io.WriteString(w, body)
}

// awsEnv gives the AWS SDK static test credentials and keeps it away from
// the machine's real AWS configuration and instance metadata.
func awsEnv(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIDTESTONLY")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_ENDPOINT_URL", "")
	t.Setenv("AWS_ENDPOINT_URL_S3", "")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "credentials"))
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
}

func TestGetS3SignsRequests(t *testing.T) {
	awsEnv(t)
	fake := &fakeS3{objects: map[string]string{"/releases/agent/agent.txt": "hello world"}}
	srv := httptest.NewServer(fake)
	defer srv.Close()

	body, size, err := getS3(context.Background(), "releases", "agent/agent.txt", s3Options{endpoint: srv.URL, region: "eu-west-1"})
	require.NoError(t, err)
	b, err := io.ReadAll(body)
	require.NoError(t, err)
	body.Close()
	assert.Equal(t, "hello world", string(b))
	assert.Equal(t, int64(len("hello world")), size)

	require.Len(t, fake.auth, 1)
	// Path-style addressing, signed with SigV4 for the given key and region.
	assert.Equal(t, "/releases/agent/agent.txt", fake.paths[0])
	assert.True(t, strings.HasPrefix(fake.auth[0], "AWS4-HMAC-SHA256 Credential=AKIDTESTONLY/"), fake.auth[0])
	assert.Contains(t, fake.auth[0], "/eu-west-1/s3/aws4_request")
}

func TestGetS3DefaultsRegion(t *testing.T) {
	awsEnv(t)
	fake := &fakeS3{objects: map[string]string{"/b/k": "x"}}
	srv := httptest.NewServer(fake)
	defer srv.Close()

	_, _, err := getS3(context.Background(), "b", "k", s3Options{endpoint: srv.URL})
	require.NoError(t, err)
	assert.Contains(t, fake.auth[0], "/"+defaultS3Region+"/s3/aws4_request")
}

func TestGetS3Errors(t *testing.T) {
	awsEnv(t)
	fake := &fakeS3{objects: map[string]string{}}
	srv := httptest.NewServer(fake)
	defer srv.Close()

	_, _, err := getS3(context.Background(), "releases", "missing.txt", s3Options{endpoint: srv.URL})
	assert.ErrorContains(t, err, "failed to download s3://releases/missing.txt")
	assert.ErrorContains(t, err, "NoSuchKey")
}

func s3Kwargs(pairs ...string) []starlark.Tuple {
	var out []starlark.Tuple
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, starlark.Tuple{starlark.String(pairs[i]), starlark.String(pairs[i+1])})
	}
	return out
}

func TestRunS3(t *testing.T) {
	awsEnv(t)
	fake := &fakeS3{objects: map[string]string{"/releases/agent/agent.txt": "hello world"}}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	fsys := afero.NewMemMapFs()
	a := &s3DownloadAction{&downloadAction{httpClient: http.DefaultClient, fsys: fsys, fn: "s3_download"}}
	run := func(pairs ...string) (*base.Result, error) {
		return a.Run(context.Background(), "", "fetch", nil, nil, s3Kwargs(pairs...))
	}

	r, err := run("bucket", "releases", "key", "agent/agent.txt", "save_to", "agent.txt", "sha256", helloSHA256, "endpoint", srv.URL)
	require.NoError(t, err)
	assert.True(t, r.Changed)
	got, err := afero.ReadFile(fsys, "agent.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(got))

	// Present with the right hash: nothing is fetched again.
	r, err = run("bucket", "releases", "key", "agent/agent.txt", "save_to", "agent.txt", "sha256", helloSHA256, "endpoint", srv.URL)
	require.NoError(t, err)
	assert.False(t, r.Changed)
	assert.Len(t, fake.paths, 1)

	// The sha256 pin applies to S3 downloads too.
	_, err = run("bucket", "releases", "key", "agent/agent.txt", "save_to", "other.txt", "sha256", strings.Repeat("0", 64), "endpoint", srv.URL)
	assert.ErrorContains(t, err, "expected sha256 hash")

	_, err = run("bucket", "", "key", "agent/agent.txt", "save_to", "x.txt", "sha256", helloSHA256)
	assert.ErrorContains(t, err, `s3_download(label="fetch"): bucket must not be empty`)

	// --store-only messages name s3_download, not download.
	a.storeOnly = true
	_, err = run("bucket", "releases", "key", "agent/agent.txt", "save_to", "x.txt", "sha256", helloSHA256, "endpoint", srv.URL)
	assert.ErrorContains(t, err, `s3_download(label="fetch"): save_to is disabled by --store-only`)
}
