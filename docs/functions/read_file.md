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
[docs/fixtures/read_file/](../fixtures/read_file).

```scrut
$ cp -R "$TESTDIR/../fixtures/read_file/." "$TMPDIR" && cd "$TMPDIR"
```

### Read a text file

<!-- fixture: ../fixtures/read_file/read.star -->
```python
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

<!-- fixture: ../fixtures/read_file/settings.json -->
```json
{"channel": "beta", "hosts": ["a", "b"]}
```

<!-- fixture: ../fixtures/read_file/json_config.star -->
```python
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

<!-- fixture: ../fixtures/read_file/home.star -->
```python
load("starcm", "read_file")

print(read_file("~/notes.txt", label = "read from home").value.strip())
```

```scrut
$ HOME="$TMPDIR" starcm home.star
line one
line two
```

## See also

- [json](json.md), [sha256](sha256.md), [list_files](list_files.md).
- [tests/read_file.md](../../tests/read_file.md) for missing files.
