package shell

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	base "github.com/discentem/starcm/functions/base"
	shelllib "github.com/discentem/starcm/libraries/shell"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

// skipWithoutSh skips tests that run sh, which Windows only has with Git Bash.
func skipWithoutSh(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not found")
	}
}

// runExec runs exec(cmd, args, extra...) through the shell action.
func runExec(t *testing.T, cmd string, args []string, extra ...starlark.Tuple) (*base.Result, error) {
	t.Helper()
	var list []starlark.Value
	for _, a := range args {
		list = append(list, starlark.String(a))
	}
	kwargs := []starlark.Tuple{
		{starlark.String("cmd"), starlark.String(cmd)},
		{starlark.String("args"), starlark.NewList(list)},
	}
	kwargs = append(kwargs, extra...)
	kwargs = append(kwargs, starlark.Tuple{starlark.String("label"), starlark.String("test")})
	a := &shellAction{executor: &shelllib.RealExecutor{}}
	return base.NewModule(context.Background(), "exec", execArgs, a).Call(nil, nil, kwargs)
}

func TestExec_CheckOffKeepsSuccessFalse(t *testing.T) {
	skipWithoutSh(t)
	res, err := runExec(t, "sh", []string{"-c", "echo oops >&2; exit 3"})
	require.NoError(t, err)
	require.False(t, res.Success)
	require.Equal(t, "oops\n", res.Return.(starlark.String).GoString())
}

func TestExec_CheckOnExpectedCodePasses(t *testing.T) {
	skipWithoutSh(t)
	res, err := runExec(t, "sh", []string{"-c", "exit 3"},
		starlark.Tuple{starlark.String("expected_exit_code"), starlark.MakeInt(3)},
		starlark.Tuple{starlark.String("check"), starlark.True},
	)
	require.NoError(t, err)
	require.True(t, res.Success)
}

func TestExec_CheckOnUnexpectedCodeStopsRun(t *testing.T) {
	skipWithoutSh(t)
	res, err := runExec(t, "sh", []string{"-c", "echo oops >&2; exit 3"},
		starlark.Tuple{starlark.String("check"), starlark.True},
	)
	require.Nil(t, res)
	require.EqualError(t, err, "exec(label=\"test\"): sh exited with status 3, expected 0\noops")
}

func TestExec_CheckOnUnexpectedCodeWithoutOutput(t *testing.T) {
	skipWithoutSh(t)
	_, err := runExec(t, "sh", []string{"-c", "exit 3"},
		starlark.Tuple{starlark.String("check"), starlark.True},
	)
	require.EqualError(t, err, "exec(label=\"test\"): sh exited with status 3, expected 0")
}

func TestExec_CheckOnMissingCommandStopsRun(t *testing.T) {
	res, err := runExec(t, "starcm-no-such-command", nil,
		starlark.Tuple{starlark.String("check"), starlark.True},
	)
	require.Nil(t, res)
	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), "exec(label=\"test\"): "), err.Error())
}

func TestExec_CheckOffMissingCommandIsNotSuccess(t *testing.T) {
	res, err := runExec(t, "starcm-no-such-command", nil)
	require.NoError(t, err)
	require.False(t, res.Success)
	require.NotEmpty(t, res.Error.Error())
}

func TestExec_Cwd(t *testing.T) {
	skipWithoutSh(t)
	if runtime.GOOS == "windows" {
		t.Skip("Git Bash's pwd prints /c/... paths")
	}
	dir := t.TempDir()
	res, err := runExec(t, "sh", []string{"-c", "pwd -P"},
		starlark.Tuple{starlark.String("cwd"), starlark.String(dir)},
	)
	require.NoError(t, err)
	want, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	require.Equal(t, filepath.ToSlash(want), strings.TrimSpace(res.Return.(starlark.String).GoString()))
}

func TestExec_RelativeCwdRefused(t *testing.T) {
	_, err := runExec(t, "true", nil,
		starlark.Tuple{starlark.String("cwd"), starlark.String("sub")},
	)
	require.ErrorContains(t, err, `exec(label="test"): cwd: "sub" is not an absolute path`)
}
