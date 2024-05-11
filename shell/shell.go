package shell

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"go.starlark.net/starlark"
)

type NopBufferCloser struct {
	*bytes.Buffer
}

func (b *NopBufferCloser) Close() error {
	return nil
}

func Shellout(ex RealExecutor, buf *bytes.Buffer) (
	starlarkhelpers.StarlarkBuiltin,
	error) {
	var wc io.WriteCloser
	if buf != nil {
		wc = &NopBufferCloser{Buffer: buf}
	} else {
		wc = nil
	}

	return func(thread *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		// exec's function signature
		var (
			name    string
			cmd     string
			cmdArgs *starlark.List
			notIf   starlark.Bool
			after   starlark.Callable
		)
		if err := starlark.UnpackArgs(
			"exec",
			args,
			kwargs,
			"cmd", &cmd,
			"args", &cmdArgs,
			"name?", &name,
			"not_if?", &notIf,
			"after?", &after,
		); err != nil {
			return starlark.None, err
		}

		if notIf.Truth() {
			fmt.Printf("skipping '%s [%s]' because not_if was true\n", cmd, cmdArgs)
			return starlark.None, nil
		}

		if after != nil {
			if _, err := starlark.Call(thread, after, args, kwargs); err != nil {
				return nil, err
			}
		}

		iter := cmdArgs.Iterate()
		defer iter.Done()
		var v starlark.Value
		var cmdArgsGo []string
		for iter.Next(&v) {
			if v.Type() != "string" {
				continue
			}
			cmdArgsGo = append(cmdArgsGo, v.(starlark.String).GoString())
		}
		fmt.Println(cmd, cmdArgsGo)
		ex.Command(cmd, cmdArgsGo...)
		var err error
		if wc == nil {
			err = ex.Stream()
		} else {
			err = ex.Stream(wc)
		}
		if err != nil {
			return nil, err
		}

		return starlark.String(buf.String()), nil
	}, nil

}

type Executor interface {
	Command(path string, args ...string)
	Stream(posters ...io.WriteCloser) error
}

var (
	_ = Executor(&RealExecutor{})
)

type RealExecutor struct {
	*exec.Cmd
}

func (e *RealExecutor) Command(bin string, args ...string) {
	e.Cmd = exec.Command(bin, args...)
}

func (e *RealExecutor) Stream(posters ...io.WriteCloser) error {
	stdout, err := e.StdoutPipe()
	if err != nil {
		fmt.Println("error here")
		return err
	}
	stderr, err := e.StderrPipe()
	if err != nil {
		return err
	}
	inputPipes := []io.ReadCloser{stdout, stderr}

	if err := e.Start(); err != nil {
		return err
	}

	if posters == nil {
		posters = []io.WriteCloser{os.Stdout}
	}

	for _, pipe := range inputPipes {
		for _, post := range posters {
			//nolint:errcheck
			go WriteOutput(pipe, post)
		}
	}

	if err := e.Wait(); err != nil {
		return err
	}
	return nil
}

func WriteOutput(in io.ReadCloser, post io.WriteCloser) error {
	r := bufio.NewScanner(in)
	for r.Scan() {
		m := r.Text()
		_, err := post.Write([]byte(m + "\n"))
		if err != nil {
			return err
		}
	}
	return nil
}
