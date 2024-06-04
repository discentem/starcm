package download

import (
	"context"

	"fmt"

	"io"

	"net/http"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/starlark-helpers"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

type action struct {
	httpClient http.Client
	fsys       afero.Fs
}

func (a *action) Run(ctx context.Context, moduleName string, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	if a.fsys == nil {
		return nil, fmt.Errorf("fsys must be provided to download module")
	}

	idx, err := starlarkhelpers.FindValueOfKeyInKwargs(kwargs, "url")
	if err != nil {
		return nil, err
	}
	if idx == starlarkhelpers.IndexNotFound {
		return nil, fmt.Errorf("'url' was not found in kwargs")
	}

	s, _, _, err := starlarkhelpers.Unquote(kwargs[idx][1].String())
	if err != nil {
		return nil, err
	}

	idx, err = starlarkhelpers.FindValueOfKeyInKwargs(kwargs, "save_to")
	if err != nil {
		return nil, err
	}
	if idx == starlarkhelpers.IndexNotFound {
		return nil, fmt.Errorf("'save_to' was not found in kwargs")
	}
	savePath, _, _, err := starlarkhelpers.Unquote(kwargs[idx][1].String())
	if err != nil {
		return nil, err
	}
	resp, err := a.httpClient.Get(s)
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
	err = afero.WriteFile(a.fsys, savePath, b, 0644)
	if err != nil {
		return nil, err
	}

	return &base.Result{
		Output: func() *string {
			s := fmt.Sprintf("downloaded file to %s", savePath)
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
	)

	return base.NewModule(
		ctx,
		"download",
		[]base.ArgPair{
			{Key: "url", Type: &str},
			{Key: "save_to", Type: &savePath},
		},
		&action{
			httpClient: httpClient,
			fsys:       fsys,
		},
	)
}
