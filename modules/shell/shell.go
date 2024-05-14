package shell

import (
	"bytes"
	"fmt"
	"io"

	shelllib "github.com/discentem/starcm/libraries/shell"
	base "github.com/discentem/starcm/modules/base"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"go.starlark.net/starlark"
)

type action struct {
	executor shelllib.Executor
	buff     bytes.Buffer
}

func (a *action) Run(args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	idx, err := starlarkhelpers.FindValueOfKeyInKwargs(kwargs, "cmd")
	if err != nil {
		return nil, err
	}
	if idx == shelllib.IndexNotFound {
		return nil, fmt.Errorf("'cmd' was not found in kwargs")
	}

	fmt.Println(kwargs[idx][1])
	c, _, _, err := starlarkhelpers.Unquote(kwargs[idx][1].String())
	if err != nil {
		return nil, err
	}

	idx, err = starlarkhelpers.FindValueOfKeyInKwargs(kwargs, "args")
	if err != nil {
		return nil, err
	}
	if idx == shelllib.IndexNotFound {
		return nil, fmt.Errorf("'args' was not found in kwargs")
	}

	fmt.Println(kwargs[idx][1])
	cargs := (kwargs[idx][1]).(*starlark.List)

	iter := cargs.Iterate()
	defer iter.Done()
	var v starlark.Value
	var cmdArgsGo []string
	for iter.Next(&v) {
		cmdArgsGo = append(cmdArgsGo, v.(starlark.String).GoString())
	}

	a.executor.Command(c, cmdArgsGo...)

	err = a.executor.Stream(&shelllib.NopBufferCloser{Buffer: &a.buff})
	if err != nil {
		return nil, err
	}
	res := &base.Result{
		Output: func() *string {
			s := a.buff.String()
			return &s
		}(),
		Success: a.executor.ExitCode() == 0,
		Changed: true,
		Diff:    nil,
		Comment: "",
	}

	return res, nil
}

func New(ex shelllib.Executor, wc io.WriteCloser) *base.Module {
	var str string
	var args *starlark.List
	return base.NewModule(
		"shell",
		[]base.ArgPair{
			{Key: "cmd", Type: &str},
			{Key: "args", Type: &args},
		},
		&action{
			executor: ex,
			buff:     bytes.Buffer{},
		},
	)
}
