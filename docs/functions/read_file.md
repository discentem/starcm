# read_file

[← All functions](../README.md)

Use `read_file` to get a file's content as a string, for example to decode a
JSON settings file with [json](json.md) or to check what's on disk before
changing it.

## Syntax

```python
load("starcm", "read_file")

read_file("settings.json", label = "read settings").value
read_file(path = "settings.json", label = "read settings").value
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `path` | string | required | The file to read, positionally or as `path =`. A leading `~` expands to the home directory; other relative paths are relative to the directory starcm runs in. |

`read_file` also takes the [common arguments](../common-functionality.md#common-arguments):
`label` (required), `only_if`, `not_if`, `timeout`, `working_directory` and
`what_if`. Only `only_if` and `not_if` affect it.

## Returns

A [starcm_result](../common-functionality.md#starcm_result) with `success =
True`, `changed = False` and `value` set to the file's content as a string. A
file that can't be read stops the run.

## Examples

Each example runs in a scratch copy of
[examples/read_file/](../../examples/read_file).

```scrut
$ cp -R "$TESTDIR/../../examples/read_file/." "$TMPDIR" && cd "$TMPDIR"
```

### Read a text file

```python
# examples/read_file/read.star
load("starcm", "read_file")

r = read_file("notes.txt", label = "read notes")
print("success=%s changed=%s" % (r.success, r.changed))
print(repr(r.value))
print(r.value.splitlines())
```

```scrut
$ starcm read.star
success=True changed=False
"line one\nline two\n"
["line one", "line two"]
```

### Read JSON settings

`examples/read_file/settings.json`:

```json
{"channel": "beta", "hosts": ["a", "b"]}
```

```python
# examples/read_file/json_config.star
load("starcm", "json", "read_file")

settings = json.decode(read_file(path = "settings.json", label = "read settings").value)
print("channel=%s hosts=%s" % (settings["channel"], settings["hosts"]))
```

```scrut
$ starcm json_config.star
channel=beta hosts=["a", "b"]
```

### Read from the home directory

Here `HOME` points at the scratch directory, which has a `notes.txt`.

```python
# examples/read_file/home.star
load("starcm", "read_file")

print(read_file("~/notes.txt", label = "read from home").value.strip())
```

```scrut
$ HOME="$TMPDIR" starcm home.star
line one
line two
```

## Errors and edge cases

These are here mostly as tests: CI runs them to check error messages and
behaviour that the examples above don't show. They run in the copy of
[examples/read_file/edge/](../../examples/read_file/edge).

```scrut
$ cd "$TMPDIR/edge"
```

### A missing file stops the run

```python
# examples/read_file/edge/missing.star
load("starcm", "read_file")
read_file("missing.txt", label = "missing")
```

```scrut
$ starcm missing.star 2>&1
Traceback (most recent call last):
  missing.star:2:10: in <toplevel>
Error in read_file: failed to read file "missing\.txt": open .*[/\\]missing\.txt: (no such file or directory|The system cannot find the file specified\.) (regex)
[1]
```

## See also

- [json](json.md), [sha256](sha256.md), [list_files](list_files.md).
