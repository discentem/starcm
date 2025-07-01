package download

import (
	"context"
	"time"

	"fmt"

	"io"

	"net/http"

	"github.com/discentem/starcm/functions/base"
	sha256lib "github.com/discentem/starcm/libraries/sha256"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

type downloadAction struct {
	httpClient *http.Client
	fsys       afero.Fs
	output     io.Writer
}

func (a *downloadAction) Run(ctx context.Context, workingDirectory string, moduleName string, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	if a.fsys == nil {
		return nil, fmt.Errorf("fsys must be provided to download module")
	}

	s, err := starlarkhelpers.FindValueinKwargs(kwargs, "url")
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, fmt.Errorf("url must be provided to download module, cannot be nil")
	}
	savePath, err := starlarkhelpers.FindValueinKwargs(kwargs, "save_to")
	if err != nil {
		return nil, err
	}
	if savePath == nil {
		return nil, fmt.Errorf("save_to must be provided to download module, cannot be nil")
	}

	resp, err := a.httpClient.Get(*s)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download file %q: %s", *s, resp.Status)
	}

	totalSize := resp.ContentLength
	f, err := a.fsys.Create(*savePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	liveProgress, err := starlarkhelpers.FindBoolInKwargs(kwargs, "live_progress", false)
	if err != nil {
		return nil, fmt.Errorf("failed to find live_progress in kwargs: %w", err)
	}

	var dest io.Writer = f

	if liveProgress {
		pw := &progressWriter{
			total: totalSize,
			out:   a.output,
			name:  *s,
		}
		if pw.out == nil {
			pw.out = io.Discard
		}

		dest = io.MultiWriter(f, pw)
	}
	_, err = io.Copy(dest, resp.Body)
	if liveProgress && a.output != nil {
		fmt.Fprintln(a.output)
	}
	if err != nil {
		return nil, err
	}

	actualHash, err := sha256lib.FromReader(f)
	if err != nil {
		return nil, err
	}

	expectedHash, err := starlarkhelpers.FindStringInKwargs(kwargs, "sha256")
	if err != nil {
		return nil, fmt.Errorf("failed to find sha256 in kwargs: %v", err)
	}

	if expectedHash != nil && *expectedHash != actualHash {
		return nil, fmt.Errorf("expected sha256 hash %s, got %s", *expectedHash, actualHash)
	}

	return &base.Result{
		Name: &moduleName,
		Output: func() *string {
			s := fmt.Sprintf("downloaded file to %s", *savePath)
			return &s
		}(),
		Success: true,
		Changed: true,
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
