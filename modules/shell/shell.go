package shell

import (
	"bytes"
	"fmt"

	"github.com/discentem/starcm/libraries/logging"
	shelllib "github.com/discentem/starcm/libraries/shell"
	base "github.com/discentem/starcm/modules/base"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/google/deck"
	"go.starlark.net/starlark"
)

type action struct {
	executor shelllib.Executor
	buff     bytes.Buffer
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

	iter := cargs.Iterate()
	defer iter.Done()
	var v starlark.Value
	var cmdArgsGo []string
	for iter.Next(&v) {
		cmdArgsGo = append(cmdArgsGo, v.(starlark.String).GoString())
	}

	a.executor.Command(c, cmdArgsGo...)

	err = a.executor.Stream()

	res := &base.Result{
		Output: func() *string {
			s := a.buff.String()
			return &s
		}(),
		Error: err,
		Success: func() bool {
			expectedExitCode, ok := expectedExitCode.Int64()
			if !ok {
				logging.Message{
					Prefix:    moduleName,
					Format:    "expectedExitCode.Int64() conversion failed: %v",
					Vs:        []any{expectedExitCode},
					Attribute: deck.V(2),
				}.Errorf()
				return false
			}
			logging.Message{
				Prefix:    moduleName,
				Format:    "expectedExitCode: %v",
				Vs:        []any{expectedExitCode},
				Attribute: deck.V(2),
			}.Infof()

			actualExitCode, err := a.executor.ExitCode()
			if err != nil {
				logging.Message{
					Prefix:    moduleName,
					Format:    "error getting exit code: %v",
					Vs:        []any{err},
					Attribute: deck.V(2),
				}.Errorf()
				return false
			}

			logging.Message{
				Prefix:    moduleName,
				Format:    "actualExitCode: %v",
				Vs:        []any{expectedExitCode},
				Attribute: deck.V(2),
			}.Infof()

			return int64(actualExitCode) == expectedExitCode
		}(),
		Changed: true,
		Diff:    nil,
		Comment: "",
	}

	return res, nil
}

func New(ex shelllib.Executor) *base.Module {
	var str string
	var args *starlark.List
	var exitCode starlark.Int
	return base.NewModule(
		"shell",
		[]base.ArgPair{
			{Key: "cmd", Type: &str},
			{Key: "args", Type: &args},
			{Key: "expected_exit_code??", Type: &exitCode},
		},
		&action{
			executor: ex,
			buff:     bytes.Buffer{},
		},
	)
}
