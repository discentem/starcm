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
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

// Fetcher downloads files from HTTP(S) URLs or S3 objects, checks them
// against a sha256 or a signature, and saves them to a path or into the
// local store. download(), s3_download() and load_remote() are thin
// Starlark adapters over it; Go code calls it directly.
type Fetcher struct {
	httpClient *http.Client
	fsys       afero.Fs
	output     io.Writer
	// store opens the local store, for downloads without SaveTo.
	store func() (*localstore.Store, error)
	// storeOnly refuses SaveTo, so every download goes into the store.
	storeOnly bool
	// keyring is --trusted-keys, for downloads pinned by signature. Nil
	// means Signature can't be used.
	keyring *signing.Keyring
}

// Request describes one download. Exactly one of SHA256 or Signature pins
// what it must match.
type Request struct {
	// Function and Label name the download in errors and messages, e.g.
	// download(label="agent"). Function defaults to "download".
	Function string
	Label    string
	// SaveTo is the path to save to; "" saves into the local store.
	SaveTo string
	// FileName names the stored file; "" uses the source's base name.
	FileName string
	SHA256   string
	// Signature locates the signature file: a URL for FetchURL, a key in
	// the same bucket for FetchS3.
	Signature    string
	LiveProgress bool
	// Folder fetches a folder with FetchFolder. The URL's path or S3 key
	// must end in "/" exactly when Folder is set; see checkFolder.
	Folder bool
}

// Fetched is a finished download.
type Fetched struct {
	// Path is where the file is: SaveTo, or its place in the store.
	Path    string
	Changed bool
	Message string
	// Manifest is a folder's manifest (FetchFolder); empty for one file.
	Manifest signing.Manifest
}

// S3Object is an object to fetch with FetchS3.
type S3Object struct {
	Bucket, Key string
	// Endpoint and Region are optional; see s3_download().
	Endpoint, Region string
}

func (r Request) errorf(format string, args ...any) error {
	fn := r.Function
	if fn == "" {
		fn = "download"
	}
	return fmt.Errorf("%s(label=%q): "+format, append([]any{fn, r.Label}, args...)...)
}

// FetchURL downloads url as req describes. With req.Folder, url names a
// folder, fetched with FetchFolder.
func (f *Fetcher) FetchURL(ctx context.Context, url string, req Request) (Fetched, error) {
	if f.fsys == nil {
		return Fetched{}, fmt.Errorf("fsys must be provided to download module")
	}
	u, err := neturl.Parse(url)
	if err != nil {
		return Fetched{}, req.errorf("%v", err)
	}
	if err := checkFolder(req, "url", url, u.Path); err != nil {
		return Fetched{}, err
	}
	if req.Folder {
		return f.FetchFolder(ctx, Source{URL: url}, req)
	}
	expectedHash, sig, err := f.pin(ctx, req, func(ctx context.Context, sigURL string) (io.ReadCloser, error) {
		body, _, err := f.httpGet(ctx, sigURL)
		return body, err
	})
	if err != nil {
		return Fetched{}, err
	}
	get := func(ctx context.Context) (io.ReadCloser, error) {
		body, size, err := f.httpGet(ctx, url)
		if err != nil {
			return nil, err
		}
		return f.withProgress(body, size, url, req.LiveProgress), nil
	}
	return f.save(ctx, req, path.Base(u.Path), expectedHash, sig, get)
}

// downloadAction is download(): it reads the call's arguments into a
// Request and runs it with the Fetcher.
type downloadAction struct {
	f *Fetcher
}

var _ base.Action = (*downloadAction)(nil)

func (a *downloadAction) Execute(inst *base.FunctionInstance) (*base.Result, error) {
	url, _ := base.Get[string](inst, "url")
	req, err := requestFrom(inst, "download")
	if err != nil {
		return nil, err
	}
	got, err := a.f.FetchURL(inst.Ctx, url, req)
	if err != nil {
		return nil, err
	}
	return resultFrom(inst.Label, got), nil
}

// requestFrom reads the arguments download() and s3_download() share.
// save_to is an output, so it must be absolute.
func requestFrom(inst *base.FunctionInstance, fn string) (Request, error) {
	saveTo, _, err := inst.AbsPath("save_to")
	if err != nil {
		return Request{}, err
	}
	fileName, _ := base.Get[string](inst, "file_name")
	sha, _ := base.Get[string](inst, "sha256")
	sig, _ := base.Get[string](inst, "signature")
	live, _ := base.Get[bool](inst, "live_progress")
	folder, _ := base.Get[bool](inst, "folder")
	return Request{
		Folder:       folder,
		Function:     fn,
		Label:        inst.Label,
		SaveTo:       saveTo,
		FileName:     fileName,
		SHA256:       sha,
		Signature:    sig,
		LiveProgress: live,
	}, nil
}

// resultFrom turns a finished download into download()'s result. value is
// the file's absolute path.
func resultFrom(label string, got Fetched) *base.Result {
	msg := got.Message
	return &base.Result{Label: label, Message: &msg, Success: true, Changed: got.Changed, Return: starlark.String(got.Path)}
}

// maxSignatureSize bounds a fetched signature file, which is one short line.
const maxSignatureSize = 64 << 10

// pin returns the sha256 a download must match: req.SHA256, or the one
// recorded in the signature req.Signature names, which fetchSig fetches and
// the --trusted-keys keyring must verify. sig is that signature file's
// content, or nil when pinned by sha256.
func (f *Fetcher) pin(ctx context.Context, req Request, fetchSig func(context.Context, string) (io.ReadCloser, error)) (sha256 string, sig []byte, err error) {
	hasHash, hasSig := req.SHA256 != "", req.Signature != ""
	switch {
	case hasHash && hasSig:
		return "", nil, req.errorf("pass sha256 or signature, not both")
	case hasHash:
		return req.SHA256, nil, nil
	case !hasSig:
		return "", nil, req.errorf("sha256 or signature must be provided")
	}
	if f.keyring == nil {
		return "", nil, req.errorf("signature needs trusted keys: run starcm with --trusted-keys")
	}
	body, err := fetchSig(ctx, req.Signature)
	if err != nil {
		return "", nil, req.errorf("fetching signature: %w", err)
	}
	defer body.Close()
	sig, err = io.ReadAll(io.LimitReader(body, maxSignatureSize+1))
	if err != nil {
		return "", nil, req.errorf("fetching signature: %w", err)
	}
	if len(sig) > maxSignatureSize {
		return "", nil, req.errorf("signature %s is larger than %d bytes", req.Signature, maxSignatureSize)
	}
	verified, err := f.keyring.VerifyDigest(sig)
	if err != nil {
		return "", nil, req.errorf("signature %s: %w", req.Signature, err)
	}
	return verified.SHA256, sig, nil
}

// save finishes a download: to req.SaveTo when it's set, otherwise into the
// local store under req.FileName or defaultName.
func (f *Fetcher) save(ctx context.Context, req Request, defaultName, expectedHash string, sig []byte, get func(context.Context) (io.ReadCloser, error)) (Fetched, error) {
	if req.SaveTo == "" {
		name := defaultName
		if req.FileName != "" {
			name = req.FileName
		}
		return f.toStore(ctx, req, name, expectedHash, sig, get)
	}
	if f.storeOnly {
		return Fetched{}, req.errorf("save_to is disabled by --store-only; omit it to download into the store")
	}
	if !filepath.IsAbs(req.SaveTo) {
		return Fetched{}, req.errorf("save_to %q is not an absolute path", req.SaveTo)
	}
	return f.toPath(ctx, req, expectedHash, sig, get)
}

// toPath downloads to req.SaveTo. The data goes to a temporary file next to
// it and is renamed into place once its sha256 matches, so SaveTo never
// holds unverified data.
func (f *Fetcher) toPath(ctx context.Context, req Request, expectedHash string, sig []byte, get func(context.Context) (io.ReadCloser, error)) (Fetched, error) {
	got, err := f.fileToPath(ctx, req, expectedHash, get)
	if err != nil || sig == nil {
		return got, err
	}
	// Keep the signature next to the file, where --trusted-keys and
	// verify() look for it.
	sigPath := req.SaveTo + signing.SignatureSuffix
	if old, err := afero.ReadFile(f.fsys, sigPath); err == nil && bytes.Equal(old, sig) {
		return got, nil
	}
	if err := afero.WriteFile(f.fsys, sigPath, sig, 0o644); err != nil {
		return Fetched{}, req.errorf("writing %s: %w", sigPath, err)
	}
	got.Changed = true
	return got, nil
}

// fileToPath is toPath without the signature.
func (f *Fetcher) fileToPath(ctx context.Context, req Request, expectedHash string, get func(context.Context) (io.ReadCloser, error)) (Fetched, error) {
	savePath := req.SaveTo
	if _, err := f.fsys.Stat(savePath); err == nil {
		existingHash, err := starcmsha256.File(f.fsys, savePath)
		if err != nil {
			return Fetched{}, fmt.Errorf("failed to hash existing file %q: %w", savePath, err)
		}
		if existingHash == expectedHash {
			return Fetched{Path: savePath, Message: fmt.Sprintf("%q already present and sha256 verified", savePath)}, nil
		}
		// Exists but wrong hash: re-download; the rename below replaces it.
	} else if !errors.Is(err, os.ErrNotExist) {
		return Fetched{}, fmt.Errorf("failed to stat %q: %w", savePath, err)
	}

	body, err := get(ctx)
	if err != nil {
		return Fetched{}, err
	}
	defer body.Close()

	dir := filepath.Dir(savePath)
	if err := f.fsys.MkdirAll(dir, 0o755); err != nil {
		return Fetched{}, fmt.Errorf("failed to create directory for %q: %w", savePath, err)
	}
	tmp, err := afero.TempFile(f.fsys, dir, "."+filepath.Base(savePath)+".tmp-")
	if err != nil {
		return Fetched{}, err
	}
	tmpName := tmp.Name()
	// Every path below closes tmp before this runs, and before the rename:
	// Windows can't delete or rename a file that is still open.
	defer f.fsys.Remove(tmpName) // no-op once renamed

	hasher := starcmsha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, hasher), body); err != nil {
		tmp.Close()
		return Fetched{}, err
	}
	f.endProgress(req.LiveProgress)
	if err := tmp.Close(); err != nil {
		return Fetched{}, err
	}
	if actualHash := starcmsha256.Hex(hasher); actualHash != expectedHash {
		return Fetched{}, fmt.Errorf("expected sha256 hash %s, got %s", expectedHash, actualHash)
	}
	// Temp files are created 0600; give the download the usual file mode.
	if err := f.fsys.Chmod(tmpName, 0o644); err != nil {
		return Fetched{}, err
	}
	if err := f.fsys.Rename(tmpName, savePath); err != nil {
		return Fetched{}, err
	}
	return Fetched{Path: savePath, Changed: true, Message: fmt.Sprintf("downloaded file to %s", savePath)}, nil
}

// toStore downloads into the local store as name and returns its path there.
func (f *Fetcher) toStore(ctx context.Context, req Request, name, expectedHash string, sig []byte, get func(context.Context) (io.ReadCloser, error)) (Fetched, error) {
	if f.store == nil {
		return Fetched{}, req.errorf("save_to must be provided; no store is configured")
	}
	if name == "" || name == "." || name == "/" {
		return Fetched{}, req.errorf("can't name the stored file; pass file_name")
	}

	store, err := f.store()
	if err != nil {
		return Fetched{}, err
	}
	stored, ok, err := store.Get(expectedHash, name)
	if err != nil {
		return Fetched{}, err
	}
	if ok {
		got := Fetched{Path: stored, Message: fmt.Sprintf("%q already in the store and sha256 verified", stored)}
		return f.storeSignature(store, req, expectedHash, name, sig, got)
	}

	body, err := get(ctx)
	if err != nil {
		return Fetched{}, err
	}
	defer body.Close()
	stored, err = store.Put(expectedHash, name, body)
	f.endProgress(req.LiveProgress)
	if err != nil {
		return Fetched{}, err
	}
	got := Fetched{Path: stored, Changed: true, Message: fmt.Sprintf("downloaded file to %s", stored)}
	return f.storeSignature(store, req, expectedHash, name, sig, got)
}

// storeSignature keeps sig, when the download was pinned by one, next to
// the stored file, so --trusted-keys can load a stored .star file.
func (f *Fetcher) storeSignature(store *localstore.Store, req Request, hash, name string, sig []byte, got Fetched) (Fetched, error) {
	if sig == nil {
		return got, nil
	}
	_, sigChanged, err := store.PutSignature(hash, name, sig)
	if err != nil {
		return Fetched{}, req.errorf("storing signature: %w", err)
	}
	if sigChanged {
		got.Changed = true
	}
	return got, nil
}

// httpGet starts an HTTP GET of url, returning the body and its size (-1
// when unknown).
func (f *Fetcher) httpGet(ctx context.Context, url string) (io.ReadCloser, int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := f.httpClient.Do(req)
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
func (f *Fetcher) withProgress(body io.ReadCloser, size int64, name string, liveProgress bool) io.ReadCloser {
	if !liveProgress {
		return body
	}
	pw := &progressWriter{
		total: size,
		out:   f.output,
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

func (f *Fetcher) endProgress(liveProgress bool) {
	if liveProgress && f.output != nil {
		fmt.Fprintln(f.output)
	}
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

// Option configures a Fetcher.
type Option func(*Fetcher)

// WithKeyring sets the --trusted-keys keyring that downloads pinned by
// signature are checked against.
func WithKeyring(k *signing.Keyring) Option {
	return func(f *Fetcher) { f.keyring = k }
}

// WithStore lets the Fetcher save into the local store when SaveTo is
// empty. open is called the first time a download needs the store.
func WithStore(open func() (*localstore.Store, error)) Option {
	return func(f *Fetcher) { f.store = open }
}

// StoreOnly makes the Fetcher refuse SaveTo, so every download goes into
// the store.
func StoreOnly(storeOnly bool) Option {
	return func(f *Fetcher) { f.storeOnly = storeOnly }
}

// WithHTTPClient sets the client for HTTP(S) downloads. The default is
// http.DefaultClient.
func WithHTTPClient(c *http.Client) Option {
	return func(f *Fetcher) { f.httpClient = c }
}

// NewFetcher returns a Fetcher that writes to fsys and prints live progress
// to output.
func NewFetcher(fsys afero.Fs, output io.Writer, opts ...Option) *Fetcher {
	f := &Fetcher{httpClient: http.DefaultClient, fsys: fsys, output: output}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// New returns the download() builtin.
//
//	download(label = "fetch agent", url = "https://example.com/agent.pkg", save_to = "/tmp/agent.pkg", sha256 = "...")
//	download(label = "fetch agent files", url = "https://example.com/agent/", folder = True, sha256 = "<manifest sha256>")
func New(ctx context.Context, f *Fetcher) *base.Module {
	return base.NewModule(
		ctx,
		"download",
		[]base.ArgPair{
			base.Arg[string]("url"),
			base.OptionalArgAllowNone[bool]("folder"),
			base.OptionalArg[string]("save_to"),
			base.OptionalArg[string]("file_name"),
			base.OptionalArg[string]("sha256"),
			base.OptionalArg[string]("signature"),
			base.OptionalArgAllowNone[bool]("live_progress"),
		},
		&downloadAction{f: f},
	)
}
