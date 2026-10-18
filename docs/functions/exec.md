# exec

[← All functions](../README.md)

Use `exec` to run a command. Its output (stdout and stderr together) is
returned as the result's `value`, and its exit code decides `success`. A
command that exits with an unexpected code doesn't stop the run: check
`success` and decide what to do, or call `fail()`.

## Syntax

```python
load("starcm", "exec")

exec(
    label              = "install agent",   # required
    cmd                = "installer",       # required
    args               = ["-pkg", "agent.pkg", "-target", "/"],  # required, may be []
    expected_exit_code = 0,
    live_output        = False,
)
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `cmd` | string | required | The program to run, found on `PATH` or given as a path. It's run directly, not through a shell. |
| `args` | list of strings | required | Arguments to pass. Use `[]` for none. |
| `expected_exit_code` | int | `0` | The exit code that counts as success. |
| `live_output` | bool | `False` | Also stream the command's output to starcm's stdout as it runs. |

`exec` also takes the [common arguments](../common-functionality.md#common-arguments): `label`
(required), `only_if`, `not_if`, `timeout`, `working_directory` and `what_if`.
`timeout` gives up waiting for the command after the given duration.
`working_directory` and `what_if` have no effect: the command runs in the
directory starcm runs in, every time.

### Files from the local store

When `cmd` or any of `args` is a path inside the
[local store](download.md#the-local-store), `exec` hashes that file again
right before the command starts and stops the run if it no longer matches the
hash it was stored under. So a file [download](download.md) checked can't be
swapped between the download and the command that uses it.

## Returns

A [starcm_result](../common-functionality.md#starcm_result):

| Field | Value |
| --- | --- |
| `success` | `True` when the exit code equals `expected_exit_code` |
| `changed` | Always `True`: starcm can't tell whether a command changed anything |
| `value` | The command's stdout and stderr, as a string |
| `error` | `"<nil>"` for exit code 0, otherwise e.g. `"exit status 3"` (even when that code was expected), or `"context deadline exceeded"` after a `timeout` |
| `message` | `""` |

Log lines about skipped `exec` calls call it `shell(label=...)`, its internal name.

## Examples

Each example runs in a scratch copy of
[docs/fixtures/exec/](../fixtures/exec).

```scrut
$ cp -R "$TESTDIR/../fixtures/exec/." "$TMPDIR" && cd "$TMPDIR"
```

### Capture a command's output

<!-- fixture: ../fixtures/exec/capture.star -->
```python
load("starcm", "exec")

r = exec(label = "say hello", cmd = "echo", args = ["hello", "world"])
print(r)
```

```scrut
$ starcm capture.star
starcm_result(changed = True, error = "<nil>", label = "say hello", message = "", success = True, value = "hello world\n")
```

### Use a shell for pipes and redirects

<!-- fixture: ../fixtures/exec/shell.star -->
```python
load("starcm", "exec")

# cmd is run directly, not through a shell. Ask for one to use pipes,
# redirects or globs.
r = exec(
    label = "count lines",
    cmd   = "sh",
    args  = ["-c", "printf 'a\\nb\\nc\\n' | wc -l | tr -d ' '"],
)
print("lines: %s" % r.value.strip())
```

```scrut
$ starcm shell.star
lines: 3
```

On Windows, use `cmd = "cmd.exe", args = ["/c", ...]` or
`cmd = "powershell.exe", args = ["-Command", ...]` the same way.

### Expect a non-zero exit code

<!-- fixture: ../fixtures/exec/exit_codes.star -->
```python
load("starcm", "exec")

ok = exec(label = "grep finds nothing", cmd = "sh", args = ["-c", "exit 1"], expected_exit_code = 1)
print("expected 1: success=%s error=%s" % (ok.success, ok.error))

bad = exec(label = "unexpected failure", cmd = "sh", args = ["-c", "echo oops >&2; exit 3"])
print("unexpected: success=%s error=%s value=%r" % (bad.success, bad.error, bad.value))
if not bad.success:
    print("handle the failure here, or stop the run with fail()")
```

```scrut
$ starcm exit_codes.star
expected 1: success=True error=exit status 1
unexpected: success=False error=exit status 3 value="oops\n"
handle the failure here, or stop the run with fail()
```

### Stream output while it runs

<!-- fixture: ../fixtures/exec/live_output.star -->
```python
load("starcm", "exec")

exec(
    label       = "long running installer",
    cmd         = "sh",
    args        = ["-c", "echo step 1; echo step 2"],
    live_output = True,
)
```

```scrut
$ starcm live_output.star
step 1
step 2
```

### Give up after a timeout

<!-- fixture: ../fixtures/exec/timeout.star -->
```python
load("starcm", "exec")

r = exec(label = "hangs", cmd = "sleep", args = ["2"], timeout = "500ms")
print("success=%s error=%s" % (r.success, r.error))
```

```scrut
$ starcm timeout.star
success=False error=context deadline exceeded
```

### Run a command only when a check fails

<!-- fixture: ../fixtures/exec/chain.star -->
```python
load("starcm", "exec")

check = exec(label = "is the tool installed?", cmd = "sh", args = ["-c", "command -v not-a-real-tool"])
exec(
    label   = "install the tool",
    cmd     = "echo",
    args    = ["installing not-a-real-tool"],
    not_if  = check.success,
    live_output = True,
)
```

```scrut
$ starcm chain.star
installing not-a-real-tool
```

### Run a file from the local store

The examples use a store in the scratch directory. On Windows, starcm refuses
a store anywhere under the CI runners' `D:\` drive (where scrut's scratch
directory lives), because the drive is owned by another account, so there the
store goes under `%LocalAppData%`:

```scrut
$ store="$TMPDIR/store"; case "$(uname -s)" in MINGW*|MSYS*|CYGWIN*) store_root=$(mktemp -d "$(cygpath -u "$LOCALAPPDATA")/starcm-docs.XXXXXX") && store="$store_root/store" ;; esac
```

<!-- fixture: ../fixtures/exec/stored.star -->
```python
load("starcm", "download", "exec")

agent = download(
    label  = "fetch agent",
    url    = "http://localhost:8000/docs/fixtures/download/files/agent-1.2.0.txt",
    sha256 = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
)

# agent.value is a path in the store, so exec() re-hashes it before running.
exec(label = "install agent", cmd = "cat", args = [agent.value], live_output = True)
```

```scrut
$ starcm --store-dir "$store" stored.star
agent v1.2.0
```

If the stored file changes after `download` checked it, `exec` refuses to run:

<!-- fixture: ../fixtures/exec/stored_changed.star -->
```python
load("starcm", "download", "exec")

agent = download(
    label  = "fetch agent",
    url    = "http://localhost:8000/docs/fixtures/download/files/agent-1.2.0.txt",
    sha256 = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
)

# Something changes the stored file after download() checked it...
exec(label = "tamper", cmd = "sh", args = ["-c", "chmod u+w \"$1\" && echo malware >> \"$1\"", "sh", agent.value])

# ...so exec() refuses to run with it.
exec(label = "install agent", cmd = "cat", args = [agent.value], live_output = True)
```

```scrut
$ starcm --store-dir "$store" stored_changed.star 2>&1
Traceback (most recent call last):
  stored_changed.star:13:5: in <toplevel>
Error in exec: exec(label="install agent"): *agent-1.2.0.txt in the store has sha256 *; it was changed after it was stored (glob)
[1]
```

```scrut
$ rm -rf "$store" ${store_root:+"$store_root"}
```

## See also

- [download](download.md#the-local-store) for the local store.
- [Common functionality](../common-functionality.md) for `only_if`, `not_if` and `timeout`.
- [tests/exec.md](../../tests/exec.md) for argument errors.
