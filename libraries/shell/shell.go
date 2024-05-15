package shell

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"os/exec"
)

const (
	IndexNotFound int = -1
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
	ExitCode() int
}

var (
	_ = Executor(&RealExecutor{})
)

type RealExecutor struct {
	*exec.Cmd
	exitCode int
}

func (e *RealExecutor) ExitCode() int {
	return e.exitCode
}

func (e *RealExecutor) Command(bin string, args ...string) {
	e.exitCode = -1
	e.Cmd = exec.Command(bin, args...)
}

func (e *RealExecutor) Stream(posters ...io.WriteCloser) error {
	stdout, err := e.StdoutPipe()
	if err != nil {
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
		if exitError, ok := err.(*exec.ExitError); ok {
			e.exitCode = exitError.ExitCode()
			return exitError
		}
	}
	e.exitCode = 0
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
