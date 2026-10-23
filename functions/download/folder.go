package download

import (
	"bytes"
	"context"
	"fmt"
	"io"
	neturl "net/url"
	"path"
	"path/filepath"
	"strings"

	starcmsha256 "github.com/discentem/starcm/libraries/sha256"
	"github.com/discentem/starcm/libraries/signing"
	"github.com/spf13/afero"
)

// Source is the folder FetchFolder fetches: an HTTP(S) URL ending in "/",
// or, when S3 is set, the S3 prefix S3.Key, also ending in "/".
type Source struct {
	URL string
	S3  *S3Object
}

func (s Source) String() string {
	if s.S3 != nil {
		return fmt.Sprintf("s3://%s/%s", s.S3.Bucket, s.S3.Key)
	}
	return s.URL
}

// IsFolder reports whether a URL or S3 key names a folder: it ends in "/".
func IsFolder(urlOrKey string) bool {
	return strings.HasSuffix(urlOrKey, "/")
}

// FetchFolder downloads the folder src as req describes. req.SHA256 or
// req.Signature pins the folder's manifest, src + signing.ManifestName;
// each file the manifest lists is fetched relative to src and must have the
// sha256 it lists. Files in the folder that the manifest doesn't list are
// never fetched.
//
// Without req.SaveTo the folder is assembled in the local store, and Path is
// its directory there. With it, each listed file is written under SaveTo the
// way FetchURL writes one file, along with the manifest and its signature;
// files already there with the right sha256 aren't fetched again, and files
// the manifest doesn't list are left alone.
func (f *Fetcher) FetchFolder(ctx context.Context, src Source, req Request) (Fetched, error) {
	if f.fsys == nil {
		return Fetched{}, fmt.Errorf("fsys must be provided to download module")
	}
	if req.FileName != "" {
		return Fetched{}, req.errorf("file_name doesn't apply to a folder (%s ends in /)", src)
	}
	var get func(ctx context.Context, rel string) (io.ReadCloser, int64, error)
	var getSig func(ctx context.Context, sig string) (io.ReadCloser, error)
	switch {
	case src.S3 != nil:
		obj := *src.S3
		if obj.Bucket == "" {
			return Fetched{}, req.errorf("bucket must not be empty")
		}
		if !IsFolder(obj.Key) {
			return Fetched{}, req.errorf("folder key %q must end in /", obj.Key)
		}
		get = func(ctx context.Context, rel string) (io.ReadCloser, int64, error) {
			return getS3(ctx, obj.Bucket, obj.Key+rel, obj)
		}
		getSig = func(ctx context.Context, key string) (io.ReadCloser, error) {
			body, _, err := getS3(ctx, obj.Bucket, key, obj)
			return body, err
		}
	default:
		u, err := neturl.Parse(src.URL)
		if err != nil {
			return Fetched{}, req.errorf("%v", err)
		}
		if !IsFolder(u.Path) {
			return Fetched{}, req.errorf("folder URL %s must end in /", src.URL)
		}
		get = func(ctx context.Context, rel string) (io.ReadCloser, int64, error) {
			return f.httpGet(ctx, fileURL(u, rel))
		}
		getSig = func(ctx context.Context, sigURL string) (io.ReadCloser, error) {
			body, _, err := f.httpGet(ctx, sigURL)
			return body, err
		}
	}

	manifestHash, sig, err := f.pin(ctx, req, getSig)
	if err != nil {
		return Fetched{}, err
	}
	fetchFile := func(ctx context.Context, rel string) (io.ReadCloser, error) {
		body, size, err := get(ctx, rel)
		if err != nil {
			return nil, err
		}
		return f.withProgress(body, size, src.String()+rel, req.LiveProgress), nil
	}

	if req.SaveTo == "" {
		return f.folderToStore(ctx, req, src, manifestHash, sig, fetchFile)
	}
	if f.storeOnly {
		return Fetched{}, req.errorf("save_to is disabled by --store-only; omit it to download into the store")
	}
	if !filepath.IsAbs(req.SaveTo) {
		return Fetched{}, req.errorf("save_to %q is not an absolute path", req.SaveTo)
	}
	return f.folderToPath(ctx, req, src, manifestHash, sig, fetchFile)
}

// fileURL is the URL of rel, a manifest path, inside the folder u. Each part
// is escaped, so names with spaces or % fetch the file they name.
func fileURL(u *neturl.URL, rel string) string {
	parts := strings.Split(rel, "/")
	for i, p := range parts {
		parts[i] = neturl.PathEscape(p)
	}
	f := *u
	f.Path = u.Path + rel
	f.RawPath = u.EscapedPath() + strings.Join(parts, "/")
	return f.String()
}

// fetchManifest downloads src's manifest and checks it against manifestHash.
func (f *Fetcher) fetchManifest(ctx context.Context, req Request, src Source, manifestHash string, fetch func(context.Context, string) (io.ReadCloser, error)) ([]byte, signing.Manifest, error) {
	body, err := fetch(ctx, signing.ManifestName)
	if err != nil {
		return nil, signing.Manifest{}, err
	}
	defer body.Close()
	b, err := io.ReadAll(io.LimitReader(body, signing.MaxManifestSize+1))
	f.endProgress(req.LiveProgress)
	if err != nil {
		return nil, signing.Manifest{}, err
	}
	if len(b) > signing.MaxManifestSize {
		return nil, signing.Manifest{}, req.errorf("%s%s is larger than %d bytes", src, signing.ManifestName, signing.MaxManifestSize)
	}
	if got, _ := starcmsha256.FromReader(bytes.NewReader(b)); got != manifestHash {
		return nil, signing.Manifest{}, fmt.Errorf("%s%s: expected sha256 hash %s, got %s", src, signing.ManifestName, manifestHash, got)
	}
	m, err := signing.ParseManifest(b)
	if err != nil {
		return nil, signing.Manifest{}, req.errorf("%s%s: %w", src, signing.ManifestName, err)
	}
	return b, m, nil
}

// folderToStore assembles the folder in the local store. A folder already
// assembled from this manifest is re-checked and used without fetching
// anything; otherwise only files not already stored are fetched.
func (f *Fetcher) folderToStore(ctx context.Context, req Request, src Source, manifestHash string, sig []byte, fetch func(context.Context, string) (io.ReadCloser, error)) (Fetched, error) {
	if f.store == nil {
		return Fetched{}, req.errorf("save_to must be provided; no store is configured")
	}
	store, err := f.store()
	if err != nil {
		return Fetched{}, err
	}
	folder, ok, err := store.GetFolder(manifestHash)
	if err != nil {
		return Fetched{}, err
	}
	if ok {
		sigChanged, err := store.PutFolderSignature(folder, sig)
		if err != nil {
			return Fetched{}, err
		}
		return Fetched{
			Path:     folder.Dir,
			Changed:  sigChanged,
			Manifest: folder.Manifest,
			Message:  fmt.Sprintf("%q already in the store and sha256 verified", folder.Dir),
		}, nil
	}

	manifest, m, err := f.fetchManifest(ctx, req, src, manifestHash, fetch)
	if err != nil {
		return Fetched{}, err
	}
	fetched := 0
	for _, p := range m.Paths() {
		hash, name := m.Files[p], path.Base(p)
		if _, ok, err := store.Get(hash, name); err != nil {
			return Fetched{}, err
		} else if ok {
			continue
		}
		body, err := fetch(ctx, p)
		if err != nil {
			return Fetched{}, err
		}
		_, err = store.Put(hash, name, body)
		body.Close()
		f.endProgress(req.LiveProgress)
		if err != nil {
			return Fetched{}, fmt.Errorf("%s%s: %w", src, p, err)
		}
		fetched++
	}
	folder, _, err = store.PutFolder(manifestHash, manifest, sig)
	if err != nil {
		return Fetched{}, err
	}
	return Fetched{
		Path:     folder.Dir,
		Changed:  true,
		Manifest: folder.Manifest,
		Message:  fmt.Sprintf("downloaded %d of %d files to %s", fetched, len(m.Files), folder.Dir),
	}, nil
}

// folderToPath writes each listed file under req.SaveTo, then the manifest
// and, when pinned by one, its signature.
func (f *Fetcher) folderToPath(ctx context.Context, req Request, src Source, manifestHash string, sig []byte, fetch func(context.Context, string) (io.ReadCloser, error)) (Fetched, error) {
	root := req.SaveTo
	manifestPath := filepath.Join(root, signing.ManifestName)
	// The manifest already there, when it's this one, says what to check;
	// otherwise fetch it.
	var manifest []byte
	var m signing.Manifest
	if b, err := afero.ReadFile(f.fsys, manifestPath); err == nil {
		if got, _ := starcmsha256.FromReader(bytes.NewReader(b)); got == manifestHash {
			if parsed, err := signing.ParseManifest(b); err == nil {
				manifest, m = b, parsed
			}
		}
	}
	if manifest == nil {
		var err error
		if manifest, m, err = f.fetchManifest(ctx, req, src, manifestHash, fetch); err != nil {
			return Fetched{}, err
		}
	}

	written := 0
	for _, p := range m.Paths() {
		fileReq := req
		fileReq.SaveTo = filepath.Join(root, filepath.FromSlash(p))
		got, err := f.fileToPath(ctx, fileReq, m.Files[p], func(ctx context.Context) (io.ReadCloser, error) {
			return fetch(ctx, p)
		})
		if err != nil {
			return Fetched{}, fmt.Errorf("%s%s: %w", src, p, err)
		}
		if got.Changed {
			written++
		}
	}
	changed := written > 0
	for _, extra := range []struct {
		path string
		data []byte
	}{
		{manifestPath, manifest},
		{manifestPath + signing.SignatureSuffix, sig},
	} {
		if extra.data == nil {
			continue
		}
		if old, err := afero.ReadFile(f.fsys, extra.path); err == nil && bytes.Equal(old, extra.data) {
			continue
		}
		if err := f.fsys.MkdirAll(root, 0o755); err != nil {
			return Fetched{}, err
		}
		if err := afero.WriteFile(f.fsys, extra.path, extra.data, 0o644); err != nil {
			return Fetched{}, req.errorf("writing %s: %w", extra.path, err)
		}
		changed = true
	}
	msg := fmt.Sprintf("%q already present and sha256 verified", root)
	if changed {
		msg = fmt.Sprintf("downloaded %d of %d files to %s", written, len(m.Files), root)
	}
	return Fetched{Path: root, Changed: changed, Manifest: m, Message: msg}, nil
}
