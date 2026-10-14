package download

import (
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

	expectedHash, err := starlarkhelpers.FindStringInKwargs(kwargs, "sha256")
	if err != nil {
		return nil, fmt.Errorf("failed to find sha256 in kwargs: %v", err)
	}
	if expectedHash == nil || *expectedHash == "" {
		return nil, fmt.Errorf("sha256 must be provided to download(label=%q), cannot be nil/empty", moduleName)
	}

	liveProgress, err := starlarkhelpers.FindBoolInKwargs(kwargs, "live_progress", false)
	if err != nil {
		return nil, fmt.Errorf("failed to find live_progress in kwargs: %w", err)
	}

	if savePath == nil {
		fileName, err := optionalString(kwargs, "file_name")
		if err != nil {
			return nil, err
		}
		return a.toStore(ctx, moduleName, *url, *expectedHash, fileName, liveProgress)
	}
	if a.storeOnly {
		return nil, fmt.Errorf("download(label=%q): save_to is disabled by --store-only; omit it to download into the store", moduleName)
	}
	return a.toPath(ctx, moduleName, *url, *savePath, *expectedHash, liveProgress)
}

// toPath downloads url to savePath. The data goes to a temporary file next to
// savePath and is renamed into place once its sha256 matches, so savePath
// never holds unverified data.
func (a *downloadAction) toPath(ctx context.Context, moduleName, url, savePath, expectedHash string, liveProgress bool) (*base.Result, error) {
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

	body, err := a.get(ctx, url, liveProgress)
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

// toStore downloads url into the local store and returns its path there.
func (a *downloadAction) toStore(ctx context.Context, moduleName, rawURL, expectedHash string, fileName *string, liveProgress bool) (*base.Result, error) {
	if a.store == nil {
		return nil, fmt.Errorf("download(label=%q): save_to must be provided; no store is configured", moduleName)
	}
	name := ""
	if fileName != nil {
		name = *fileName
	} else if u, err := neturl.Parse(rawURL); err == nil {
		name = path.Base(u.Path)
	}
	if name == "" || name == "." || name == "/" {
		return nil, fmt.Errorf("download(label=%q): can't name the file from url %q; pass file_name", moduleName, rawURL)
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
		return unchanged(moduleName, fmt.Sprintf("%q already in the store and sha256 verified", stored), starlark.String(stored)), nil
	}

	body, err := a.get(ctx, rawURL, liveProgress)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	stored, err = store.Put(expectedHash, name, body)
	a.endProgress(liveProgress)
	if err != nil {
		return nil, err
	}
	return changed(moduleName, fmt.Sprintf("downloaded file to %s", stored), starlark.String(stored)), nil
}

// get starts downloading url. Reading the body prints progress when
// liveProgress is set.
func (a *downloadAction) get(ctx context.Context, url string, liveProgress bool) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to download file %q: %s", url, resp.Status)
	}
	if !liveProgress {
		return resp.Body, nil
	}
	pw := &progressWriter{
		total: resp.ContentLength,
		out:   a.output,
		name:  url,
	}
	if pw.out == nil {
		pw.out = io.Discard
	}
	return struct {
		io.Reader
		io.Closer
	}{io.TeeReader(resp.Body, pw), resp.Body}, nil
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
		_, _ = fmt.Fprintf(pw.out, "\rDownloading %s... %d%%", pw.name, (pw.written*100)/pw.total)
		pw.lastPrint = now
	}

	return n, nil
}

// Option configures download().
type Option func(*downloadAction)

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
	}
	for _, opt := range opts {
		opt(a)
	}
	return base.NewModule(
		ctx,
		"download",
		[]base.ArgPair{
			base.Arg[string]("url"),
			base.Arg[string]("save_to?"),
			base.Arg[string]("file_name?"),
			base.Arg[string]("sha256"),
			base.Arg[bool](string(starlarkhelpers.OptionalKeyword("live_progress"))),
		},
		a,
	)
}
