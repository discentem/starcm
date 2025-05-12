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
}

func (a *shellAction) Run(ctx context.Context, workingDirectory string, moduleName string, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {

	cmd, err := starlarkhelpers.FindValueinKwargs(kwargs, "cmd")
	if cmd == nil {
		return nil, fmt.Errorf("'cmd' was not found in kwargs")
	}
	if cmd == nil {
		return nil, fmt.Errorf("'cmd' was nil")
	}

	cargs, err := starlarkhelpers.FindRawValueInKwargs(kwargs, "args")
	if err != nil {
		return nil, err
	}
	cargsList, ok := cargs.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("'args' was not a list")
	}

	expectedExitCode, err := starlarkhelpers.FindIntInKwargs(kwargs, "expected_exit_code", 0)
	if err != nil {
		return nil, err
	}

	liveOutput, err := starlarkhelpers.FindBoolInKwargs(kwargs, "live_output", false)
	if err != nil {
		return nil, err
	}

	iter := cargsList.Iterate()
	defer iter.Done()
	var v starlark.Value
	var cmdArgsGo []string
	for iter.Next(&v) {
		cmdArgsGo = append(cmdArgsGo, v.(starlark.String).GoString())
	}

	ex := &shelllib.RealExecutor{}
	ex.Command(*cmd, cmdArgsGo...)

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
			Name: &moduleName,
			Output: func() *string {
				s := buff.String()
				return &s
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
			Diff:    nil,
			Comment: "",
		}
	}()

	select {
	case <-ctx.Done():
		select {
		case res := <-resultChan:
			return res, ctx.Err()
		default:
			return &base.Result{
				Name:  &moduleName,
				Error: ctx.Err(),
				Output: func() *string {
					s := buff.String()
					return &s
				}(),
			}, ctx.Err()
		}
	case res := <-resultChan:
		return res, err
	}
}

func New(ctx context.Context, executor shelllib.Executor) *base.Module {
	var (
		str        string
		args       *starlark.List
		exitCode   starlark.Int
		liveOutput starlark.Bool
	)

	return base.NewModule(
		ctx,
		"shell",
		[]base.ArgPair{
			{Key: "cmd", Type: &str},
			{Key: "args", Type: &args},
			{Key: "expected_exit_code??", Type: &exitCode},
			{Key: "live_output??", Type: &liveOutput},
		},
		&shellAction{
			executor: executor,
		},
	)
}
