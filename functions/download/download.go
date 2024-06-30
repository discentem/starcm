package download

import (
	"context"

	"fmt"

	"io"

	"net/http"

	"github.com/discentem/starcm/functions/base"
	sha256lib "github.com/discentem/starcm/libraries/sha256"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

type action struct {
	httpClient *http.Client
	fsys       afero.Fs
}

func (a *action) Run(ctx context.Context, workingDirectory string, moduleName string, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
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
		return nil, fmt.Errorf("failed to download file: %s", resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	f, err := a.fsys.Create(*savePath)
	if err != nil {
		return nil, err
	}
	actualHash, err := sha256lib.FromReader(f)
	if err != nil {
		return nil, err
	}

	expectedHash, err := starlarkhelpers.FindValueinKwargs(kwargs, "sha256")
	if err != nil {
		return nil, err
	}

	if expectedHash != nil && *expectedHash != actualHash {
		return nil, fmt.Errorf("expected sha256 hash %s, got %s", *expectedHash, actualHash)
	}
	defer f.Close()
	_, err = f.Write(b)
	if err != nil {
		return nil, err
	}

	return &base.Result{
		Output: func() *string {
			s := fmt.Sprintf("downloaded file to %s", *savePath)
			return &s
		}(),
		Success: true,
		Changed: true,
	}, nil
}

func New(ctx context.Context, httpClient http.Client, fsys afero.Fs) *base.Module {
	var (
		str      string
		savePath string
		sha256   string
	)

	return base.NewModule(
		ctx,
		"download",
		[]base.ArgPair{
			{Key: "url", Type: &str},
			{Key: "save_to", Type: &savePath},
			{Key: "sha256", Type: &sha256},
		},
		&action{
			httpClient: &httpClient,
			fsys:       fsys,
		},
	)
}
