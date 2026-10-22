# list_files

[← All functions](../README.md)

Use `list_files` to find files under a directory, recursively, for example to
sign, hash or [inspect](inspect.md) every `.star` file in a configuration
repository.

## Syntax

```python
load("starcm", "here", "list_files")

list_files(here("config"), pattern = "*.star", label = "find config").value
list_files(dir = "/etc/agent", pattern = "*.conf", label = "find settings").value
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `dir` | string | required | The absolute directory to search, positionally or as `dir =`. |
| `pattern` | string | `"*"` | A [glob](https://pkg.go.dev/path#Match) matched against each file's name (not its path): `*`, `?`, `[abc]`. An invalid pattern fails the run. |

`list_files` also takes the [common arguments](../common-functionality.md#common-arguments):
`label` (required), `only_if`, `not_if`, `timeout` and
`what_if`. Only `only_if` and `not_if` affect it.

## Returns

A [starcm_result](../common-functionality.md#starcm_result) with `success =
True`, `changed = False` and `value` set to a sorted list of file paths
relative to `dir`, with `/` separators on every platform. Directories aren't
listed.

## Examples

Each example runs in a scratch copy of
[examples/list_files/](../../examples/list_files), which has:

```scrut
$ cp -R "$TESTDIR/../../examples/list_files/." "$TMPDIR" && cd "$TMPDIR"
```

```text
config/
├── lib/
│   ├── macos.star
│   └── windows.star
├── main.star
└── notes.txt
```

### List files by pattern

```python
# examples/list_files/list.star
load("starcm", "list_files", "here")

print(list_files(here("config"), label = "everything").value)
print(list_files(here("config"), pattern = "*.star", label = "star files").value)
print(list_files(dir = here("config/lib"), pattern = "w*", label = "windows only").value)
```

```scrut
$ starcm list.star
["lib/macos.star", "lib/windows.star", "main.star", "notes.txt"]
["lib/macos.star", "lib/windows.star", "main.star"]
["windows.star"]
```

### Build a manifest of hashes

```python
# examples/list_files/manifest.star
load("starcm", "list_files", "sha256", "here")

for f in list_files(here("config"), pattern = "*.star", label = "find config").value:
    h = sha256(here("config", f), label = "hash " + f).value
    print("%s  %s" % (h[:12], f))
```

```scrut
$ starcm manifest.star
cbdeb467b4cb  lib/macos.star
76708f9814ef  lib/windows.star
2377a494b815  main.star
```

## Errors and edge cases

These are here mostly as tests: CI runs them to check error messages and
behaviour that the examples above don't show. They run in the copy of
[examples/list_files/edge/](../../examples/list_files/edge).

```scrut
$ cd "$TMPDIR/edge"
```

### An invalid pattern stops the run

```python
# examples/list_files/edge/bad_pattern.star
load("starcm", "list_files", "here")
list_files(here("tree"), pattern = "[", label = "bad pattern")
```

```scrut
$ starcm bad_pattern.star 2>&1
Traceback (most recent call last):
  bad_pattern.star:2:11: in <toplevel>
Error in list_files: invalid pattern "[": syntax error in pattern
[1]
```

## See also

- [sign](sign.md) has an example that signs every `.star` file `list_files` finds.
- [inspect](inspect.md) has an example that checks every file against a policy.
