package shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	base "github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/libraries/logging"
	shelllib "github.com/discentem/starcm/libraries/shell"
	"github.com/google/deck"
	"go.starlark.net/starlark"
)

type shellAction struct {
	executor shelllib.Executor
	// checkStored verifies a cmd or arg that names a file in the local
	// store, right before the command starts. Nil means no store.
	checkStored func(path string) error
}

var _ base.Action = (*shellAction)(nil)

func (a *shellAction) Execute(inst *base.FunctionInstance) (*base.Result, error) {
	ctx, moduleName := inst.Ctx, inst.Label
	c, _ := base.Get[string](inst, "cmd")
	// A cmd with a path separator names a script bundled next to the .star
	// file; a bare name is looked up on PATH.
	if strings.ContainsAny(c, `/\`) || strings.HasPrefix(c, "~") {
		p, _, err := inst.ResourcePath("cmd")
		if err != nil {
			return nil, err
		}
		c = p
	}
	cargs, _ := base.Get[*starlark.List](inst, "args")
	cwd, _, err := inst.AbsPath("cwd")
	if err != nil {
		return nil, err
	}
	exitCode, _ := base.Get[starlark.Int](inst, "expected_exit_code")
	expectedExitCode, ok := exitCode.Int64()
	if !ok {
		return nil, fmt.Errorf("exec(label=%q): expected_exit_code %s is out of range", moduleName, exitCode)
	}
	liveOutput, _ := base.Get[starlark.Bool](inst, "live_output")
	check, _ := base.Get[starlark.Bool](inst, "check")

	var cmdArgsGo []string
	for i := 0; i < cargs.Len(); i++ {
		s, ok := starlark.AsString(cargs.Index(i))
		if !ok {
			return nil, fmt.Errorf("exec(label=%q): args[%d]: got %s, want string", moduleName, i, cargs.Index(i).Type())
		}
		cmdArgsGo = append(cmdArgsGo, s)
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
	ex.Dir = cwd

	buff := bytes.NewBuffer(nil)
	wc := shelllib.NopBufferCloser{
		Buffer: buff,
	}

	posters := []io.WriteCloser{&wc}
	if bool(liveOutput) {
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

				// The command never started, so there is no exit code.
				var exitErr *exec.ExitError
				if err != nil && !errors.As(err, &exitErr) {
					return false
				}

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
		if bool(check) && !res.Success {
			return nil, checkError(moduleName, c, ex, res, expectedExitCode, buff.String())
		}
		return res, nil
	}
}

// checkError is the error that stops the run when check = True and the
// command couldn't start or exited with a code other than expected.
func checkError(label, cmd string, ex shelllib.Executor, res *base.Result, expected int64, output string) error {
	var exitErr *exec.ExitError
	if res.Error != nil && !errors.As(res.Error, &exitErr) {
		return fmt.Errorf("exec(label=%q): %w", label, res.Error)
	}
	actual, err := ex.ExitCode()
	if err != nil {
		return fmt.Errorf("exec(label=%q): %w", label, err)
	}
	msg := fmt.Sprintf("exec(label=%q): %s exited with status %d, expected %d", label, cmd, actual, expected)
	if out := strings.TrimSpace(output); out != "" {
		msg += "\n" + out
	}
	return errors.New(msg)
}

// execArgs are exec()'s own arguments.
var execArgs = []base.ArgPair{
	base.Arg[string]("cmd"),
	base.Arg[*starlark.List]("args"),
	base.OptionalArgAllowNone[starlark.Int]("expected_exit_code"),
	base.OptionalArgAllowNone[starlark.Bool]("live_output"),
	base.OptionalArgAllowNone[starlark.Bool]("check"),
	base.OptionalArg[string]("cwd"),
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
		"exec",
		execArgs,
		a,
	)
}
