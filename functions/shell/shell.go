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

type action struct{}

func (a *action) Run(ctx context.Context, moduleName string, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	idx, err := starlarkhelpers.FindValueOfKeyInKwargs(kwargs, "cmd")
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

	idx, err = starlarkhelpers.FindValueOfKeyInKwargs(kwargs, "args")
	if err != nil {
		return nil, err
	}
	if idx == starlarkhelpers.IndexNotFound {
		return nil, fmt.Errorf("'args' was not found in kwargs")
	}
	cargs := (kwargs[idx][1]).(*starlark.List)

	idx, err = starlarkhelpers.FindValueOfKeyInKwargs(kwargs, "expected_exit_code")
	if err != nil {
		return nil, err
	}
	var expectedExitCode *starlark.Int
	if idx == starlarkhelpers.IndexNotFound {
		// If no expected code is provided, assume 0
		expectedExitCode = func() *starlark.Int {
			i := starlark.MakeInt(0)
			return &i
		}()
	} else {
		expectedExitCode = func() *starlark.Int {
			i := (kwargs[idx][1]).(starlark.Int)
			return &i
		}()
	}

	idx, err = starlarkhelpers.FindValueOfKeyInKwargs(kwargs, "live_output")
	if err != nil {
		return nil, err
	}
	var liveOutput starlark.Bool
	if idx == starlarkhelpers.IndexNotFound {
		liveOutput = starlark.Bool(false)
	} else {
		liveOutput = (kwargs[idx][1]).(starlark.Bool)
	}

	iter := cargs.Iterate()
	defer iter.Done()
	var v starlark.Value
	var cmdArgsGo []string
	for iter.Next(&v) {
		cmdArgsGo = append(cmdArgsGo, v.(starlark.String).GoString())
	}

	ex := shelllib.RealExecutor{}
	ex.Command(c, cmdArgsGo...)

	buff := bytes.NewBuffer(nil)
	wc := shelllib.NopBufferCloser{
		Buffer: buff,
	}

	posters := []io.WriteCloser{&wc}
	if liveOutput.Truth() {
		posters = append(posters, os.Stdout)
	}

	resultChan := make(chan *base.Result, 1)
	logging.Log(moduleName, deck.V(2), "info", "len(posters): %v", len(posters))

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
				expectedExitCode, ok := expectedExitCode.Int64()
				if !ok {
					logging.Log(moduleName, nil, "error", "expectedExitCode.Int64() conversion failed: %v", err)
					return false
				}
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

func New(ctx context.Context) *base.Module {
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
		&action{},
	)
}
