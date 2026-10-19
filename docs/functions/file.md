# file

[← All functions](../README.md)

Use `file` to make sure a file exists with exactly the given content, or that
it doesn't exist at all. The file is only written when its content differs, so
`changed` tells you whether anything happened.

## Syntax

```python
load("starcm", "file")

file(
    label       = "agent config",   # required
    path        = "/etc/agent.conf", # required
    content     = "",
    action      = "create",          # or "delete"
    mode        = 0o644,
    create_dirs = False,
)
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `path` | string | required | The file to manage. A leading `~` expands to the home directory (`$HOME`); other relative paths are relative to the directory starcm runs in. |
| `content` | string | `""` | The file's full content (`action = "create"` only). For several lines, use a triple-quoted string; see [Multi-line content](#multi-line-content). |
| `action` | string | `"create"` | `"create"` makes the file match `content`; `"delete"` removes it. |
| `mode` | int | `0o644` | Permissions for a newly created file. An existing file's permissions are left alone. |
| `create_dirs` | bool | `False` | Create missing parent directories. Without it, a missing parent directory fails the run. |

`file` also takes the [common arguments](../common-functionality.md#common-arguments): `label`
(required), `only_if`, `not_if`, `timeout`, `working_directory` and
`what_if`. `working_directory` and `what_if` have no effect.

## Returns

A [starcm_result](../common-functionality.md#starcm_result):

| Field | Value |
| --- | --- |
| `success` | `True` |
| `changed` | `True` when the file was written or deleted |
| `message` | One of `created file "<abs path>"`, `file "<abs path>" already exists with correct content`, `deleted file "<abs path>"` or `file "<abs path>" does not exist` |
| `value` | `None` |

## Examples

Each example runs in a scratch copy of
[docs/fixtures/file/](../fixtures/file).

```scrut
$ cp -R "$TESTDIR/../fixtures/file/." "$TMPDIR" && cd "$TMPDIR"
```

### Create a file

<!-- fixture: ../fixtures/file/create.star -->
```python
load("starcm", "file")

r = file(
    label   = "agent settings",
    path    = "agent.conf",
    content = "server = https://config.example.com\n",
)
print(r)
```

```scrut
$ starcm create.star && cat agent.conf
starcm_result(changed = True, error = "<nil>", label = "agent settings", message = "created file \"*agent.conf\"", success = True, value = None) (glob)
server = https://config.example.com
```

The second run finds the right content and doesn't touch the file:

```scrut
$ starcm create.star
starcm_result(changed = False, error = "<nil>", label = "agent settings", message = "file \"*agent.conf\" already exists with correct content", success = True, value = None) (glob)
```

### Replace the content

`content` is the whole file, not a line to add.

<!-- fixture: ../fixtures/file/update.star -->
```python
load("starcm", "file")

r = file(
    label   = "agent settings",
    path    = "agent.conf",
    content = """\
server = https://config.example.com
channel = beta
""",
)
print("changed=%s" % r.changed)
```

```scrut
$ starcm update.star && cat agent.conf
changed=True
server = https://config.example.com
channel = beta
```

### Multi-line content

Write multi-line content as a triple-quoted string, as in the example above.
A backslash right after the opening `"""` skips the first newline, so the
content starts on the next line at the left margin, exactly as it will be in
the file:

```python
content = """\
server = https://config.example.com
channel = beta
""",
```

Inside a function, where the lines would be indented with the code, use
[dedent](dedent.md) to take the indentation off.

### Create parent directories and set the mode

<!-- fixture: ../fixtures/file/create_dirs.star -->
```python
load("starcm", "file")

r = file(
    label       = "agent config",
    path        = "etc/agent/agent.conf",
    content     = "server = https://config.example.com\n",
    mode        = 0o600,
    create_dirs = True,
)
print("success=%s changed=%s" % (r.success, r.changed))
```

```scrut
$ starcm create_dirs.star && cat etc/agent/agent.conf
success=True changed=True
server = https://config.example.com
```

Without `create_dirs`, a missing parent directory fails the run:

<!-- fixture: ../fixtures/file/missing_dir.star -->
```python
load("starcm", "file")

file(label = "no parent", path = "nowhere/file.txt", content = "x")
```

```scrut
$ starcm missing_dir.star 2>&1
Traceback (most recent call last):
  missing_dir.star:3:5: in <toplevel>
Error in file: directory ".*[/\\]nowhere" does not exist and create_dirs is false (regex)
[1]
```

### Write to the home directory

Here `HOME` points at the scratch directory so the example doesn't touch your
real home directory.

<!-- fixture: ../fixtures/file/home.star -->
```python
load("starcm", "file")

r = file(label = "dotfile", path = "~/.agentrc", content = "debug = false\n")
print("changed=%s" % r.changed)
```

```scrut
$ HOME="$TMPDIR" starcm home.star && cat "$TMPDIR/.agentrc"
changed=True
debug = false
```

### Delete a file

Deleting a file that's already gone succeeds with `changed = False`.

<!-- fixture: ../fixtures/file/delete.star -->
```python
load("starcm", "file")

for i in range(2):
    r = file(label = "remove agent settings", path = "agent.conf", action = "delete")
    print("pass %d: changed=%s message=%s" % (i + 1, r.changed, r.message))
```

```scrut
$ starcm delete.star
pass 1: changed=True message=deleted file "*agent.conf" (glob)
pass 2: changed=False message=file "*agent.conf" does not exist (glob)
```

## See also

- [template](template.md) to render a file's content from a template.
- [json](json.md) to write JSON content.
- [tests/file.md](../../tests/file.md).
