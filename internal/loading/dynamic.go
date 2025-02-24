package loading

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/libraries/logging"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/google/deck"
	"go.starlark.net/starlark"
)

type loadDynamicAction struct {
	thread *starlark.Thread
}

func (a *loadDynamicAction) Run(ctx context.Context, workingDirectory string, moduleName string, args starlark.Tuple, kwargs []starlark.Tuple) (base.ActionReturn, error) {
	var InModulePath *string
	var modulePath string
	var absolutePath bool

	InModulePath, err := starlarkhelpers.FindValueinKwargs(kwargs, "path")
	if err != nil {
		return nil, fmt.Errorf("failed to find 'path' in kwargs: %v", err)
	}
	if InModulePath == nil {
		return nil, fmt.Errorf("path must be provided to load_dynamic module, cannot be nil")
	}
	modulePath = *InModulePath

	idx, err := starlarkhelpers.FindIndexOfValueInKwargs(kwargs, "absolute_path")
	if err == nil {
		InAbsolutePath, err := starlarkhelpers.FindBoolValueFromIndexInKwargs(kwargs, idx)
		if err != nil {
			return nil, fmt.Errorf("failed to find 'absolute_path' in kwargs: %v", err)
		}
		absolutePath = *InAbsolutePath
	} else {
		absolutePath = false
	}
	if !absolutePath {
		if len(a.thread.CallStack()) > 0 {
			dirName := filepath.Dir(a.thread.CallStack().At(1).Pos.Filename())
			modulePath = filepath.Join(dirName, modulePath)
		}
	}
	logging.Log(moduleName, deck.V(3), "info", "Loading module from path: %q", modulePath)
	module, err := a.thread.Load(a.thread, modulePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load module from path %q: %v", modulePath, err)
	}

	dict := starlark.NewDict(len(module))
	for key, val := range module {
		err = dict.SetKey(starlark.String(key), val)
		if err != nil {
			return nil, err
		}
	}
	dict.Freeze()
	return &base.RawStarlarkValueReturn{Value: dict}, nil
}

func New(ctx context.Context, thread *starlark.Thread) (*base.Module, error) {
	var (
		path         string
		absolutePath string
	)

	if thread == nil {
		return nil, fmt.Errorf("thread cannot be nil, must pass a valid starlark thread")
	}

	return base.NewModule(
		ctx,
		"load_dynamic",
		[]base.ArgPair{
			{Key: "path", Type: &path},
			{Key: "absolute_path??", Type: &absolutePath},
		},
		&loadDynamicAction{
			thread: thread,
		},
	), nil
}
