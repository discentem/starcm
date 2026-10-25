package shell

import (
	"context"
	"errors"
	"os"
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

// envDict builds exec()'s env argument from alternating keys and values.
func envDict(t *testing.T, kv ...starlark.Value) starlark.Tuple {
	t.Helper()
	d := starlark.NewDict(len(kv) / 2)
	for i := 0; i < len(kv); i += 2 {
		require.NoError(t, d.SetKey(kv[i], kv[i+1]))
	}
	return starlark.Tuple{starlark.String("env"), d}
}

// output is the command's output, without the newline exec adds after its
// last line.
func output(t *testing.T, res *base.Result) string {
	t.Helper()
	return strings.TrimSuffix(res.Return.(starlark.String).GoString(), "\n")
}

func TestExec_EnvAddsAndOverrides(t *testing.T) {
	skipWithoutSh(t)
	t.Setenv("STARCM_TEST_KEPT", "kept")
	t.Setenv("STARCM_TEST_OVERRIDDEN", "old")
	res, err := runExec(t, "sh", []string{"-c", `printf '%s %s %s' "$STARCM_TEST_NEW" "$STARCM_TEST_KEPT" "$STARCM_TEST_OVERRIDDEN"`},
		envDict(t,
			starlark.String("STARCM_TEST_NEW"), starlark.String("new"),
			starlark.String("STARCM_TEST_OVERRIDDEN"), starlark.String("override"),
		),
	)
	require.NoError(t, err)
	require.True(t, res.Success)
	require.Equal(t, "new kept override", output(t, res))
}

// env adds to the environment rather than replacing it, so PATH (used to
// find sh and the commands it runs) is still there.
func TestExec_EnvKeepsPath(t *testing.T) {
	skipWithoutSh(t)
	res, err := runExec(t, "sh", []string{"-c", `command -v sh >/dev/null && printf '%s' "$CGO_ENABLED"`},
		envDict(t, starlark.String("CGO_ENABLED"), starlark.String("0")),
	)
	require.NoError(t, err)
	require.True(t, res.Success, output(t, res))
	require.Equal(t, "0", output(t, res))
}

func TestExec_EnvEmptyValue(t *testing.T) {
	skipWithoutSh(t)
	t.Setenv("STARCM_TEST_EMPTIED", "full")
	res, err := runExec(t, "sh", []string{"-c", `printf '[%s]' "${STARCM_TEST_EMPTIED-unset}"`},
		envDict(t, starlark.String("STARCM_TEST_EMPTIED"), starlark.String("")),
	)
	require.NoError(t, err)
	require.Equal(t, "[]", output(t, res))
}

func TestExec_EnvOmittedOrNoneOrEmptyInherits(t *testing.T) {
	skipWithoutSh(t)
	t.Setenv("STARCM_TEST_KEPT", "kept")
	script := []string{"-c", `printf '%s' "$STARCM_TEST_KEPT"`}
	for name, extra := range map[string][]starlark.Tuple{
		"omitted": nil,
		"None":    {{starlark.String("env"), starlark.None}},
		"{}":      {envDict(t)},
	} {
		res, err := runExec(t, "sh", script, extra...)
		require.NoError(t, err, name)
		require.Equal(t, "kept", output(t, res), name)
	}
}

func TestExec_EnvWithCwdAndCheck(t *testing.T) {
	skipWithoutSh(t)
	dir := t.TempDir()
	_, err := runExec(t, "sh", []string{"-c", `printf '%s' "$STARCM_TEST_MSG"; test -f marker`},
		envDict(t, starlark.String("STARCM_TEST_MSG"), starlark.String("no marker here")),
		starlark.Tuple{starlark.String("cwd"), starlark.String(dir)},
		starlark.Tuple{starlark.String("check"), starlark.True},
	)
	require.EqualError(t, err, "exec(label=\"test\"): sh exited with status 1, expected 0\nno marker here")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "marker"), nil, 0o644))
	res, err := runExec(t, "sh", []string{"-c", `printf '%s' "$STARCM_TEST_MSG"; test -f marker`},
		envDict(t, starlark.String("STARCM_TEST_MSG"), starlark.String("found")),
		starlark.Tuple{starlark.String("cwd"), starlark.String(dir)},
		starlark.Tuple{starlark.String("check"), starlark.True},
	)
	require.NoError(t, err)
	require.Equal(t, "found", output(t, res))
}

// The store check still sees cmd and every arg when env is given, and still
// stops the command.
func TestExec_EnvStoreCheck(t *testing.T) {
	var checked []string
	a := &shellAction{
		executor: &shelllib.RealExecutor{},
		checkStored: func(p string) error {
			checked = append(checked, p)
			if p == "tampered" {
				return errors.New("tampered in the store has changed")
			}
			return nil
		},
	}
	kwargs := []starlark.Tuple{
		{starlark.String("cmd"), starlark.String("starcm-no-such-command")},
		{starlark.String("args"), starlark.NewList([]starlark.Value{starlark.String("tampered")})},
		envDict(t, starlark.String("A"), starlark.String("b")),
		{starlark.String("label"), starlark.String("test")},
	}
	_, err := base.NewModule(context.Background(), "exec", execArgs, a).Call(nil, nil, kwargs)
	require.EqualError(t, err, `exec(label="test"): tampered in the store has changed`)
	require.Equal(t, []string{"starcm-no-such-command", "tampered"}, checked)
}

func TestExec_EnvErrors(t *testing.T) {
	for _, tc := range []struct {
		env  starlark.Tuple
		want string
	}{
		{envDict(t, starlark.String("A"), starlark.MakeInt(1)), `exec(label="test"): env["A"]: got int, want string`},
		{envDict(t, starlark.MakeInt(1), starlark.String("x")), `exec(label="test"): env: key 1: got int, want string`},
		{envDict(t, starlark.String(""), starlark.String("x")), `exec(label="test"): env: invalid variable name ""`},
		{envDict(t, starlark.String("A=B"), starlark.String("x")), `exec(label="test"): env: invalid variable name "A=B"`},
		{envDict(t, starlark.String("A"), starlark.String("x\x00y")), `exec(label="test"): env["A"]: value holds a NUL byte`},
	} {
		res, err := runExec(t, "starcm-no-such-command", nil, tc.env)
		require.Nil(t, res)
		require.EqualError(t, err, tc.want)
	}
	_, err := runExec(t, "starcm-no-such-command", nil, starlark.Tuple{starlark.String("env"), starlark.NewList(nil)})
	require.ErrorContains(t, err, "env")
}
