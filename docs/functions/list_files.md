# list_files

[← All functions](../README.md)

Use `list_files` to find files under a directory, recursively, for example to
sign, hash or [inspect](inspect.md) every `.star` file in a configuration
repository.

## Syntax

```python
load("starcm", "list_files")

list_files("config", pattern = "*.star", label = "find config").value
list_files(dir = "config", pattern = "*.star", label = "find config").value
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `dir` | string | `"."` | Directory to search, positionally or as `dir =`, relative to the directory starcm runs in. |
| `pattern` | string | `"*"` | A [glob](https://pkg.go.dev/path#Match) matched against each file's name (not its path): `*`, `?`, `[abc]`. An invalid pattern fails the run. |

`list_files` also takes the [common arguments](../common-functionality.md#common-arguments):
`label` (required), `only_if`, `not_if`, `timeout`, `working_directory` and
`what_if`. Only `only_if` and `not_if` affect it.

## Returns

A [starcm_result](../common-functionality.md#starcm_result) with `success =
True`, `changed = False` and `value` set to a sorted list of file paths
relative to `dir`, with `/` separators on every platform. Directories aren't
listed.

## Examples

Each example runs in a scratch copy of
[docs/fixtures/list_files/](../fixtures/list_files), which has:

```scrut
$ cp -R "$TESTDIR/../fixtures/list_files/." "$TMPDIR" && cd "$TMPDIR"
```

```scrut
$ find config -type f | sort
config/lib/macos.star
config/lib/windows.star
config/main.star
config/notes.txt
```

### List files by pattern

```scrut
$ cat list.star
load("starcm", "list_files")

print(list_files("config", label = "everything").value)
print(list_files("config", pattern = "*.star", label = "star files").value)
print(list_files(dir = "config/lib", pattern = "w*", label = "windows only").value)
```

```scrut
$ starcm list.star
["lib/macos.star", "lib/windows.star", "main.star", "notes.txt"]
["lib/macos.star", "lib/windows.star", "main.star"]
["windows.star"]
```

### Build a manifest of hashes

```scrut
$ cat manifest.star
load("starcm", "list_files", "sha256")

for f in list_files("config", pattern = "*.star", label = "find config").value:
    h = sha256("config/" + f, label = "hash " + f).value
    print("%s  %s" % (h[:12], f))
```

```scrut
$ starcm manifest.star
cbdeb467b4cb  lib/macos.star
76708f9814ef  lib/windows.star
2377a494b815  main.star
```

## See also

- [sign](sign.md) has an example that signs every `.star` file `list_files` finds.
- [inspect](inspect.md) has an example that checks every file against a policy.
- [tests/list_files.md](../../tests/list_files.md) for invalid patterns.
