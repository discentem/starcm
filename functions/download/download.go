package download

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"fmt"

	"io"

	"net/http"
	neturl "net/url"
	"path"

	"github.com/discentem/starcm/functions/base"
	localstore "github.com/discentem/starcm/libraries/local_store"
	starcmsha256 "github.com/discentem/starcm/libraries/sha256"
	"github.com/discentem/starcm/libraries/signing"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

type downloadAction struct {
	httpClient *http.Client
	fsys       afero.Fs
	output     io.Writer
	// store opens the local store, for downloads without save_to.
	store func() (*localstore.Store, error)
	// storeOnly refuses save_to, so every download goes into the store.
	storeOnly bool
	// fn is the Starlark function name used in error messages.
	fn string
	// keyring is --trusted-keys, for downloads pinned by signature. Nil
	// means signature can't be used.
	keyring *signing.Keyring
}

// Ensure downloadAction implements base.Runnable
var _ base.Runnable = (*downloadAction)(nil)

func (a *downloadAction) Run(
	ctx context.Context,
	workingDirectory string,
	moduleName string,
	thread *starlark.Thread,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (*base.Result, error) {
	if a.fsys == nil {
		return nil, fmt.Errorf("fsys must be provided to download module")
	}

	url, err := starlarkhelpers.FindValueinKwargs(kwargs, "url")
	if err != nil {
		return nil, err
	}
	if url == nil {
		return nil, fmt.Errorf("url must be provided to download module, cannot be nil")
	}

	savePath, err := optionalString(kwargs, "save_to")
	if err != nil {
		return nil, err
	}

	expectedHash, sig, err := a.pin(ctx, moduleName, kwargs, func(ctx context.Context, sigURL string) (io.ReadCloser, error) {
		body, _, err := a.httpGet(ctx, sigURL)
		return body, err
	})
	if err != nil {
		return nil, err
	}

	liveProgress, err := starlarkhelpers.FindBoolInKwargs(kwargs, "live_progress", false)
	if err != nil {
		return nil, fmt.Errorf("failed to find live_progress in kwargs: %w", err)
	}

	get := func(ctx context.Context) (io.ReadCloser, error) {
		body, size, err := a.httpGet(ctx, *url)
		if err != nil {
			return nil, err
		}
		return a.withProgress(body, size, *url, liveProgress), nil
	}
	urlName := ""
	if u, err := neturl.Parse(*url); err == nil {
		urlName = path.Base(u.Path)
	}
	return a.save(ctx, moduleName, kwargs, savePath, urlName, expectedHash, sig, get, liveProgress)
}

// maxSignatureSize bounds a fetched signature file, which is one short line.
const maxSignatureSize = 64 << 10

// pin returns the sha256 a download must match: the sha256 argument, or the
// one recorded in the signature named by the signature argument, which
// fetchSig fetches and the --trusted-keys keyring must verify. sig is that
// signature file's content, or nil when pinned by sha256.
func (a *downloadAction) pin(ctx context.Context, label string, kwargs []starlark.Tuple, fetchSig func(context.Context, string) (io.ReadCloser, error)) (sha256 string, sig []byte, err error) {
	hash, err := optionalString(kwargs, "sha256")
	if err != nil {
		return "", nil, err
	}
	sigRef, err := optionalString(kwargs, "signature")
	if err != nil {
		return "", nil, err
	}
	hasHash, hasSig := hash != nil && *hash != "", sigRef != nil && *sigRef != ""
	switch {
	case hasHash && hasSig:
		return "", nil, fmt.Errorf("%s(label=%q): pass sha256 or signature, not both", a.fn, label)
	case hasHash:
		return *hash, nil, nil
	case !hasSig:
		return "", nil, fmt.Errorf("%s(label=%q): sha256 or signature must be provided", a.fn, label)
	}
	if a.keyring == nil {
		return "", nil, fmt.Errorf("%s(label=%q): signature needs trusted keys: run starcm with --trusted-keys", a.fn, label)
	}
	body, err := fetchSig(ctx, *sigRef)
	if err != nil {
		return "", nil, fmt.Errorf("%s(label=%q): fetching signature: %w", a.fn, label, err)
	}
	defer body.Close()
	sig, err = io.ReadAll(io.LimitReader(body, maxSignatureSize+1))
	if err != nil {
		return "", nil, fmt.Errorf("%s(label=%q): fetching signature: %w", a.fn, label, err)
	}
	if len(sig) > maxSignatureSize {
		return "", nil, fmt.Errorf("%s(label=%q): signature %s is larger than %d bytes", a.fn, label, *sigRef, maxSignatureSize)
	}
	verified, err := a.keyring.VerifyDigest(sig)
	if err != nil {
		return "", nil, fmt.Errorf("%s(label=%q): signature %s: %w", a.fn, label, *sigRef, err)
	}
	return verified.SHA256, sig, nil
}

// save finishes a download: to savePath when it's given, otherwise into the
// local store under file_name or defaultName.
func (a *downloadAction) save(ctx context.Context, moduleName string, kwargs []starlark.Tuple, savePath *string, defaultName, expectedHash string, sig []byte, get func(context.Context) (io.ReadCloser, error), liveProgress bool) (*base.Result, error) {
	if savePath == nil {
		fileName, err := optionalString(kwargs, "file_name")
		if err != nil {
			return nil, err
		}
		name := defaultName
		if fileName != nil {
			name = *fileName
		}
		return a.toStore(ctx, moduleName, name, expectedHash, sig, get, liveProgress)
	}
	if a.storeOnly {
		return nil, fmt.Errorf("%s(label=%q): save_to is disabled by --store-only; omit it to download into the store", a.fn, moduleName)
	}
	return a.toPath(ctx, moduleName, *savePath, expectedHash, sig, get, liveProgress)
}

// toPath downloads url to savePath. The data goes to a temporary file next to
// savePath and is renamed into place once its sha256 matches, so savePath
// never holds unverified data.
func (a *downloadAction) toPath(ctx context.Context, moduleName, savePath, expectedHash string, sig []byte, get func(context.Context) (io.ReadCloser, error), liveProgress bool) (*base.Result, error) {
	r, err := a.fileToPath(ctx, moduleName, savePath, expectedHash, get, liveProgress)
	if err != nil || sig == nil {
		return r, err
	}
	// Keep the signature next to the file, where --trusted-keys and
	// verify() look for it.
	sigPath := savePath + signing.SignatureSuffix
	if old, err := afero.ReadFile(a.fsys, sigPath); err == nil && bytes.Equal(old, sig) {
		return r, nil
	}
	if err := afero.WriteFile(a.fsys, sigPath, sig, 0o644); err != nil {
		return nil, fmt.Errorf("%s(label=%q): writing %s: %w", a.fn, moduleName, sigPath, err)
	}
	r.Changed = true
	return r, nil
}

// fileToPath is toPath without the signature.
func (a *downloadAction) fileToPath(ctx context.Context, moduleName, savePath, expectedHash string, get func(context.Context) (io.ReadCloser, error), liveProgress bool) (*base.Result, error) {
	if _, err := a.fsys.Stat(savePath); err == nil {
		existingHash, err := starcmsha256.File(a.fsys, savePath)
		if err != nil {
			return nil, fmt.Errorf("failed to hash existing file %q: %w", savePath, err)
		}
		if existingHash == expectedHash {
			return unchanged(moduleName, fmt.Sprintf("%q already present and sha256 verified", savePath), starlark.None), nil
		}
		// Exists but wrong hash: re-download; the rename below replaces it.
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to stat %q: %w", savePath, err)
	}

	body, err := get(ctx)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	dir := filepath.Dir(savePath)
	if err := a.fsys.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory for %q: %w", savePath, err)
	}
	tmp, err := afero.TempFile(a.fsys, dir, "."+filepath.Base(savePath)+".tmp-")
	if err != nil {
		return nil, err
	}
	tmpName := tmp.Name()
	// Every path below closes tmp before this runs, and before the rename:
	// Windows can't delete or rename a file that is still open.
	defer a.fsys.Remove(tmpName) // no-op once renamed

	hasher := starcmsha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, hasher), body); err != nil {
		tmp.Close()
		return nil, err
	}
	a.endProgress(liveProgress)
	if err := tmp.Close(); err != nil {
		return nil, err
	}
	if actualHash := starcmsha256.Hex(hasher); actualHash != expectedHash {
		return nil, fmt.Errorf("expected sha256 hash %s, got %s", expectedHash, actualHash)
	}
	// Temp files are created 0600; give the download the usual file mode.
	if err := a.fsys.Chmod(tmpName, 0o644); err != nil {
		return nil, err
	}
	if err := a.fsys.Rename(tmpName, savePath); err != nil {
		return nil, err
	}
	return changed(moduleName, fmt.Sprintf("downloaded file to %s", savePath), starlark.None), nil
}

// toStore downloads into the local store as name and returns its path there.
func (a *downloadAction) toStore(ctx context.Context, moduleName, name, expectedHash string, sig []byte, get func(context.Context) (io.ReadCloser, error), liveProgress bool) (*base.Result, error) {
	if a.store == nil {
		return nil, fmt.Errorf("%s(label=%q): save_to must be provided; no store is configured", a.fn, moduleName)
	}
	if name == "" || name == "." || name == "/" {
		return nil, fmt.Errorf("%s(label=%q): can't name the stored file; pass file_name", a.fn, moduleName)
	}

	store, err := a.store()
	if err != nil {
		return nil, err
	}
	stored, ok, err := store.Get(expectedHash, name)
	if err != nil {
		return nil, err
	}
	if ok {
		r := unchanged(moduleName, fmt.Sprintf("%q already in the store and sha256 verified", stored), starlark.String(stored))
		return a.storeSignature(store, moduleName, expectedHash, name, sig, r)
	}

	body, err := get(ctx)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	stored, err = store.Put(expectedHash, name, body)
	a.endProgress(liveProgress)
	if err != nil {
		return nil, err
	}
	r := changed(moduleName, fmt.Sprintf("downloaded file to %s", stored), starlark.String(stored))
	return a.storeSignature(store, moduleName, expectedHash, name, sig, r)
}

// storeSignature keeps sig, when the download was pinned by one, next to
// the stored file, so --trusted-keys can load a stored .star file.
func (a *downloadAction) storeSignature(store *localstore.Store, moduleName, hash, name string, sig []byte, r *base.Result) (*base.Result, error) {
	if sig == nil {
		return r, nil
	}
	_, sigChanged, err := store.PutSignature(hash, name, sig)
	if err != nil {
		return nil, fmt.Errorf("%s(label=%q): storing signature: %w", a.fn, moduleName, err)
	}
	if sigChanged {
		r.Changed = true
	}
	return r, nil
}

// httpGet starts an HTTP GET of url, returning the body and its size (-1
// when unknown).
func (a *downloadAction) httpGet(ctx context.Context, url string) (io.ReadCloser, int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("failed to download file %q: %s", url, resp.Status)
	}
	return resp.Body, resp.ContentLength, nil
}

// withProgress makes reading body print progress for name when
// liveProgress is set.
func (a *downloadAction) withProgress(body io.ReadCloser, size int64, name string, liveProgress bool) io.ReadCloser {
	if !liveProgress {
		return body
	}
	pw := &progressWriter{
		total: size,
		out:   a.output,
		name:  name,
	}
	if pw.out == nil {
		pw.out = io.Discard
	}
	return struct {
		io.Reader
		io.Closer
	}{io.TeeReader(body, pw), body}
}

func (a *downloadAction) endProgress(liveProgress bool) {
	if liveProgress && a.output != nil {
		fmt.Fprintln(a.output)
	}
}

// optionalString returns the value of key in kwargs, or nil if it wasn't passed.
func optionalString(kwargs []starlark.Tuple, key string) (*string, error) {
	idx, err := starlarkhelpers.FindIndexOfValueInKwargs(kwargs, key)
	if errors.Is(err, starlarkhelpers.ErrIndexNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return starlarkhelpers.FindValueFromIndexInKwargs(kwargs, idx)
}

func changed(label, message string, value starlark.Value) *base.Result {
	return &base.Result{Label: label, Message: &message, Success: true, Changed: true, Return: value}
}

func unchanged(label, message string, value starlark.Value) *base.Result {
	return &base.Result{Label: label, Message: &message, Success: true, Changed: false, Return: value}
}

type progressWriter struct {
	name      string
	total     int64
	written   int64
	lastPrint time.Time
	out       io.Writer
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.written += int64(n)

	now := time.Now()
	if now.Sub(pw.lastPrint) > 500*time.Millisecond || pw.written == pw.total {
		if pw.total > 0 {
			_, _ = fmt.Fprintf(pw.out, "\rDownloading %s... %d%%", pw.name, (pw.written*100)/pw.total)
		} else {
			_, _ = fmt.Fprintf(pw.out, "\rDownloading %s... %d bytes", pw.name, pw.written)
		}
		pw.lastPrint = now
	}

	return n, nil
}

// Option configures download().
type Option func(*downloadAction)

// WithKeyring sets the --trusted-keys keyring that downloads pinned by
// signature are checked against.
func WithKeyring(k *signing.Keyring) Option {
	return func(a *downloadAction) { a.keyring = k }
}

// WithStore lets download() save into the local store when save_to is
// omitted. open is called the first time a download needs the store.
func WithStore(open func() (*localstore.Store, error)) Option {
	return func(a *downloadAction) { a.store = open }
}

// StoreOnly makes download() refuse save_to, so every download goes into the
// store.
func StoreOnly(storeOnly bool) Option {
	return func(a *downloadAction) { a.storeOnly = storeOnly }
}

func New(ctx context.Context, httpClient http.Client, fsys afero.Fs, writer io.Writer, opts ...Option) *base.Module {
	a := &downloadAction{
		httpClient: &httpClient,
		fsys:       fsys,
		output:     writer,
		fn:         "download",
	}
	for _, opt := range opts {
		opt(a)
	}
	return base.NewModule(
		ctx,
		"download",
		[]base.ArgPair{
			base.Arg[string]("url"),
			base.OptionalArg[string]("save_to"),
			base.OptionalArg[string]("file_name"),
			base.OptionalArg[string]("sha256"),
			base.OptionalArg[string]("signature"),
			base.OptionalArgAllowNone[bool]("live_progress"),
		},
		a,
	)
}
