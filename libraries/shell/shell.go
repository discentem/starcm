package shell

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"

	"github.com/discentem/starcm/libraries/logging"
	"github.com/google/deck"
)

type NopBufferCloser struct {
	*bytes.Buffer
}

func (b *NopBufferCloser) Close() error {
	return nil
}

type Executor interface {
	Command(path string, args ...string)
	Stream(posters ...io.WriteCloser) error
	ExitCode() (int, error)
}

var (
	_ = Executor(&RealExecutor{})
)

type RealExecutor struct {
	*exec.Cmd
}

func (e *RealExecutor) ExitCode() (int, error) {
	if !e.Cmd.ProcessState.Exited() {
		return -2, errors.New("ExitCode() called before cmd finished")
	}
	return e.Cmd.ProcessState.ExitCode(), nil
}

func (e *RealExecutor) Command(bin string, args ...string) {
	e.Cmd = exec.Command(bin, args...)
}

func (e *RealExecutor) Stream(posters ...io.WriteCloser) error {
	stdout, err := e.StdoutPipe()
	if err != nil {
		logging.Log("shelllib", deck.V(1), "error", "error getting stdout pipe: %v", err)
		return err
	}
	stderr, err := e.StderrPipe()
	if err != nil {
		logging.Log("shelllib", deck.V(1), "error", "error getting stderr pipe: %v", err)
		return err
	}
	inputPipes := []io.ReadCloser{stdout, stderr}

	if err := e.Start(); err != nil {
		logging.Log("shelllib", deck.V(1), "error", "error starting command: %v", err)
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
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError
		}
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
