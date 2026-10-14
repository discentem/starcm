package shell

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	base "github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/libraries/logging"
	shelllib "github.com/discentem/starcm/libraries/shell"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/google/deck"
	"go.starlark.net/starlark"
)

type shellAction struct {
	executor shelllib.Executor
	// checkStored verifies a cmd or arg that names a file in the local
	// store, right before the command starts. Nil means no store.
	checkStored func(path string) error
}

var _ base.Runnable = (*shellAction)(nil)

func (a *shellAction) Run(ctx context.Context, workingDirectory string, moduleName string, thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {

	idx, err := starlarkhelpers.FindIndexOfValueInKwargs(kwargs, "cmd")
	if err != nil {
		return nil, err
	}
	if idx == starlarkhelpers.IndexNotFound {
		return nil, fmt.Errorf("'cmd' was not found in kwargs")
	}

	c, _, _, err := starlarkhelpers.Unquote(kwargs[idx][1].String())
	if err != nil {
		return nil, err
	}

	idx, err = starlarkhelpers.FindIndexOfValueInKwargs(kwargs, "args")
	if err != nil {
		return nil, err
	}
	if idx == starlarkhelpers.IndexNotFound {
		return nil, fmt.Errorf("'args' was not found in kwargs")
	}
	cargs := (kwargs[idx][1]).(*starlark.List)

	expectedExitCode, err := starlarkhelpers.FindIntInKwargs(kwargs, "expected_exit_code", 0)
	if err != nil {
		return nil, err
	}

	liveOutput, err := starlarkhelpers.FindBoolInKwargs(kwargs, "live_output", false)
	if err != nil {
		return nil, err
	}

	iter := cargs.Iterate()
	defer iter.Done()
	var v starlark.Value
	var cmdArgsGo []string
	for iter.Next(&v) {
		cmdArgsGo = append(cmdArgsGo, v.(starlark.String).GoString())
	}

	if a.checkStored != nil {
		for _, p := range append([]string{c}, cmdArgsGo...) {
			if err := a.checkStored(p); err != nil {
				return nil, fmt.Errorf("exec(label=%q): %w", moduleName, err)
			}
		}
	}

	ex := &shelllib.RealExecutor{}
	ex.Command(c, cmdArgsGo...)

	buff := bytes.NewBuffer(nil)
	wc := shelllib.NopBufferCloser{
		Buffer: buff,
	}

	posters := []io.WriteCloser{&wc}
	if liveOutput {
		posters = append(posters, os.Stdout)
	}

	logging.Log(moduleName, deck.V(3), "info", "number of io.WriteClosers: %v", posters)

	resultChan := make(chan *base.Result, 1)

	go func() {
		err := ex.Stream(posters...)
		resultChan <- &base.Result{
			Label: moduleName,
			Return: func() starlark.Value {
				s := buff.String()
				return starlark.String(s)
			}(),
			Error: err,
			Success: func() bool {
				logging.Log(moduleName, deck.V(2), "info", "expectedExitCode: %v", expectedExitCode)

				actualExitCode, err := ex.ExitCode()
				if err != nil {
					logging.Log(moduleName, nil, "error", "error getting exit code: %v", err)
					return false
				}
				logging.Log(moduleName, deck.V(2), "info", "actualExitCode: %v", actualExitCode)

				return int64(actualExitCode) == expectedExitCode
			}(),
			Changed: true,
		}
	}()

	select {
	case <-ctx.Done():
		select {
		case res := <-resultChan:
			return res, ctx.Err()
		default:
			return &base.Result{
				Label: moduleName,
				Error: ctx.Err(),
				Message: func() *string {
					s := buff.String()
					return &s
				}(),
			}, ctx.Err()
		}
	case res := <-resultChan:
		return res, err
	}
}

// Option configures exec().
type Option func(*shellAction)

// WithStoreCheck makes exec() call check on cmd and every arg before the
// command starts, and refuse to run if it returns an error.
func WithStoreCheck(check func(path string) error) Option {
	return func(a *shellAction) { a.checkStored = check }
}

func New(ctx context.Context, executor shelllib.Executor, opts ...Option) *base.Module {
	a := &shellAction{
		executor: executor,
	}
	for _, opt := range opts {
		opt(a)
	}
	return base.NewModule(
		ctx,
		"shell",
		[]base.ArgPair{
			base.Arg[string]("cmd"),
			base.Arg[*starlark.List]("args"),
			base.Arg[starlark.Int]("expected_exit_code??"),
			base.Arg[starlark.Bool]("live_output??"),
		},
		a,
	)
}
