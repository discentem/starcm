# Common functionality

[← Documentation index](README.md)

Every starcm function except the [json](functions/json.md),
[vars](functions/vars.md) and [struct](functions/struct.md) helpers shares
the behaviour on this page: the same [common arguments](#common-arguments),
and the same kind of return value, a [starcm_result](#starcm_result).

The examples run in a scratch copy of
[docs/fixtures/common_functionality/](fixtures/common_functionality).

```scrut
$ cp -R "$TESTDIR/fixtures/common_functionality/." "$TMPDIR" && cd "$TMPDIR"
```

## Common arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `label` | string | required | Names the call in logs, errors and the result. |
| [only_if](#only_if-and-not_if) | bool | `True` | Run only when this is true. |
| [not_if](#only_if-and-not_if) | bool | `False` | Skip when this is true. |
| [timeout](#timeout) | string | none | Give up after this [duration](https://pkg.go.dev/time#ParseDuration), e.g. `"30s"` or `"5m"`. |
| [working_directory](#working_directory) | string | the calling file's directory | Base directory for relative paths. Only [template](functions/template.md) uses it. |
| [what_if](#what_if) | bool | `False` | Show what would change without changing it. Only [template](functions/template.md) supports it. |

Arguments apply only to the call they're passed to; nothing carries over to
later calls.

### only_if and not_if

`only_if` and `not_if` take a bool, usually from an earlier result. A
skipped call logs why and returns an empty result: `success = False`,
`changed = False`, `label = ""` and `value = None`.

<!-- fixture: fixtures/common_functionality/conditions.star -->
```python
load("starcm", "exec", "write")

installed = exec(label = "already installed?", cmd = "sh", args = ["-c", "exit 1"])

write("installing", label = "install", not_if = installed.success)
write("configuring", label = "configure", only_if = installed.success)
skipped = write("never printed", label = "skipped", only_if = False)
print("skipped: %s" % skipped)
```

```scrut
$ starcm conditions.star
installing
INFO: \d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} \[configure\]: skipping write\(label="configure"\) because only_if was false (regex)
INFO: \d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} \[skipped\]: skipping write\(label="skipped"\) because only_if was false (regex)
skipped: starcm_result(changed = False, error = "<nil>", label = "", message = "", success = False, value = None)
```

Starlark's own `if` works too, and is clearer when several calls depend on the
same condition. `only_if` / `not_if` keep the skip in the log.

### timeout

[exec](functions/exec.md) returns a failed result when its timeout expires;
[download](functions/download.md) fails the run.

<!-- fixture: fixtures/common_functionality/timeout.star -->
```python
load("starcm", "exec")

r = exec(label = "slow command", cmd = "sleep", args = ["2"], timeout = "500ms")
print("success=%s error=%s" % (r.success, r.error))
```

```scrut
$ starcm timeout.star
success=False error=context deadline exceeded
```

### working_directory

[template](functions/template.md) resolves its `template` and `destination`
paths against the directory of the `.star` file that calls it. An explicit
`working_directory` is used as given instead, and a relative one is relative
to the directory starcm runs in. Other functions resolve relative paths
against the directory starcm runs in, whatever `working_directory` says.

<!-- fixture: fixtures/common_functionality/sub/render.star -->
```python
load("starcm", "template")

# By default, template and destination are relative to this file's directory...
template(label = "next to me", template = "hello.tpl", destination = "here.txt", data = {"where": "sub"})

# ...an explicit working_directory is used as given instead.
template(
    label             = "from the current directory",
    template          = "sub/hello.tpl",
    destination       = "there.txt",
    data              = {"where": "the current directory"},
    working_directory = ".",
)
```

```scrut
$ starcm sub/render.star && cat sub/here.txt there.txt
Hello from sub.
Hello from the current directory.
```

### what_if

With `what_if = True`, [template](functions/template.md) renders and
reports `changed` as if it had written the file, but writes nothing.
Other functions accept `what_if` and ignore it, so don't rely on it for a
dry run of a whole configuration.

<!-- fixture: fixtures/common_functionality/what_if.star -->
```python
load("starcm", "template")

r = template(
    label    = "dry run",
    template = "sub/hello.tpl",
    data     = {"where": "a dry run"},
    what_if  = True,
)
print("changed=%s message=%r" % (r.changed, r.message))
```

```scrut
$ starcm what_if.star
changed=True message="Hello from a dry run.\n"
```

## starcm_result

Functions return a frozen `starcm_result` struct. Each function's page says
what its fields hold.

| Field | Type | Description |
| --- | --- | --- |
| `label` | string | The call's `label`. |
| `success` | bool | Whether the call did what it was asked. |
| `changed` | bool | Whether it changed anything on the machine. |
| `message` | string | A human-readable summary, or `""`. |
| `error` | string | The error as text, or `"<nil>"` when there was none. |
| `value` | any | The function's output, e.g. a command's output or a file's content, or `None`. |

Most problems (bad arguments, missing files, failed downloads) stop the run
with an error and a traceback instead of returning a result. A few functions
report failure in the result instead, so the configuration can decide what to
do: [exec](functions/exec.md) for unexpected exit codes and timeouts, and
[verify](functions/verify.md) for files that don't verify. Call `fail()` to
stop the run yourself.

<!-- fixture: fixtures/common_functionality/result.star -->
```python
load("starcm", "exec")

r = exec(label = "print the date", cmd = "echo", args = ["2026-10-17"])
print(r)
print("%s: success=%s changed=%s" % (r.label, r.success, r.changed))
```

```scrut
$ starcm result.star
starcm_result(changed = True, error = "<nil>", label = "print the date", message = "", success = True, value = "2026-10-17\n")
print the date: success=True changed=True
```

## See also

- [tests/common_args.md](../tests/common_args.md) checks that arguments don't leak between calls.
