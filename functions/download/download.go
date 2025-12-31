package download

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"time"

	"fmt"

	"io"

	"net/http"

	"github.com/discentem/starcm/functions/base"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

type downloadAction struct {
	httpClient *http.Client
	fsys       afero.Fs
	output     io.Writer
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

	savePath, err := starlarkhelpers.FindValueinKwargs(kwargs, "save_to")
	if err != nil {
		return nil, err
	}
	if savePath == nil {
		return nil, fmt.Errorf("save_to must be provided to download(label=%q), cannot be nil", moduleName)
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

	fileSHA256 := func(path string) (string, error) {
		f, err := a.fsys.Open(path)
		if err != nil {
			return "", err
		}
		defer f.Close()

		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return "", err
		}
		return fmt.Sprintf("%x", h.Sum(nil)), nil
	}

	if _, err := a.fsys.Stat(*savePath); err == nil {
		existingHash, err := fileSHA256(*savePath)
		if err != nil {
			return nil, fmt.Errorf("failed to hash existing file %q: %w", *savePath, err)
		}
		if existingHash == *expectedHash {
			return &base.Result{
				Label: moduleName,
				Message: func() *string {
					s := fmt.Sprintf("%q already present and sha256 verified", *savePath)
					return &s
				}(),
				Success: true,
				Changed: false,
				Return:  starlark.None,
			}, nil
		}

		// Exists but wrong hash: remove and re-download.
		if err := a.fsys.Remove(*savePath); err != nil {
			return nil, fmt.Errorf("existing file %q has wrong sha256; failed to remove: %w", *savePath, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to stat %q: %w", *savePath, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, *url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download file %q: %s", *url, resp.Status)
	}

	totalSize := resp.ContentLength

	f, err := a.fsys.Create(*savePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	hasher := sha256.New()

	var dest io.Writer = io.MultiWriter(f, hasher)
	if liveProgress {
		pw := &progressWriter{
			total: totalSize,
			out:   a.output,
			name:  *url,
		}
		if pw.out == nil {
			pw.out = io.Discard
		}
		dest = io.MultiWriter(f, hasher, pw)
	}

	if _, err := io.Copy(dest, resp.Body); err != nil {
		// Best-effort cleanup of partial download
		_ = a.fsys.Remove(*savePath)
		return nil, err
	}
	if liveProgress && a.output != nil {
		fmt.Fprintln(a.output)
	}

	actualHash := fmt.Sprintf("%x", hasher.Sum(nil))
	if *expectedHash != actualHash {
		_ = a.fsys.Remove(*savePath)
		return nil, fmt.Errorf("expected sha256 hash %s, got %s", *expectedHash, actualHash)
	}

	return &base.Result{
		Label: moduleName,
		Message: func() *string {
			s := fmt.Sprintf("downloaded file to %s", *savePath)
			return &s
		}(),
		Success: true,
		Changed: true,
		Return:  starlark.None,
	}, nil
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

func New(ctx context.Context, httpClient http.Client, fsys afero.Fs, writer io.Writer) *base.Module {
	var (
		str          string
		savePath     string
		sha256       string
		liveProgress bool
	)

	return base.NewModule(
		ctx,
		"download",
		[]base.ArgPair{
			{Key: "url", Type: &str},
			{Key: "save_to", Type: &savePath},
			{Key: "sha256", Type: &sha256},
			{
				Key:  string(starlarkhelpers.OptionalKeyword("live_progress")),
				Type: &liveProgress,
			},
		},
		&downloadAction{
			httpClient: &httpClient,
			fsys:       fsys,
			output:     writer,
		},
	)
}
