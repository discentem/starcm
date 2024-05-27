package shell

import (
	"bytes"
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

type action struct {
	executor shelllib.Executor
	buff     *bytes.Buffer
}

func (a *action) Run(moduleName string, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
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

	a.executor.Command(c, cmdArgsGo...)

	wc := shelllib.NopBufferCloser{
		Buffer: a.buff,
	}

	posters := []io.WriteCloser{&wc}
	if liveOutput.Truth() {
		posters = append(posters, os.Stdout)
	}
	err = a.executor.Stream(posters...)
	if err != nil {
		fmt.Println("error immediately after stream: ", err)
		return nil, err
	}

	res := &base.Result{
		Name: &moduleName,
		Output: func() *string {
			s := a.buff.String()
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

			actualExitCode, err := a.executor.ExitCode()
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

	return res, nil
}

func New(ex shelllib.Executor, buff *bytes.Buffer) *base.Module {
	var (
		str        string
		args       *starlark.List
		exitCode   starlark.Int
		liveOutput starlark.Bool
	)
	if buff != nil {
		buff = &bytes.Buffer{}
	}

	return base.NewModule(
		"shell",
		[]base.ArgPair{
			{Key: "cmd", Type: &str},
			{Key: "args", Type: &args},
			{Key: "expected_exit_code??", Type: &exitCode},
			{Key: "live_output??", Type: &liveOutput},
		},
		&action{
			executor: ex,
			buff:     buff,
		},
	)
}
