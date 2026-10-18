# Loading files

[← Documentation index](README.md)

starcm configuration is [Starlark](https://github.com/google/starlark-go/blob/master/doc/spec.md),
a small dialect of Python. Unlike Bazel's Starlark, starcm allows `if` and
`for` at the top level of a file. Everything starcm adds is loaded with
`load()`:

```python
load("starcm", "download", "exec", "file")   # starcm's functions
load("stdlib", "struct")                     # struct()
load("lib/users.star", "ADMINS", "greet")    # names from another file
load("starcm", run = "exec")                 # load under another name
```

| Module | Names |
| --- | --- |
| `starcm` | [download](functions/download.md), [s3_download](functions/s3_download.md), [exec](functions/exec.md), [file](functions/file.md), [template](functions/template.md), [install_mac_pkg](functions/install_mac_pkg.md), [write](functions/write.md), [date](functions/date.md), [load_dynamic](functions/load_dynamic.md), [read_file](functions/read_file.md), [list_files](functions/list_files.md), [sha256](functions/sha256.md), [inspect](functions/inspect.md), [sign](functions/sign.md), [verify](functions/verify.md), [json](functions/json.md), [vars](functions/vars.md), [temp_dir](functions/temp_dir.md) |
| `stdlib` | [struct](functions/struct.md) |
| any path ending in `.star` | the file's top-level names |

Starlark's built-ins, such as `print`, `fail`, `len`, `range`, `sorted` and
`str`, need no `load()`.

## Paths

- `//path/to/file.star` is relative to the directory starcm runs in (the
  workspace).
- An absolute path is used as is.
- Any other path is relative to the directory of the file that calls `load()`.

A module is run once per run: loading the same module string again reuses
the first result. Under
[--trusted-keys](cli.md#--trusted-keys-dir), every loaded file needs a valid
signature. To pick a file at runtime, use
[load_dynamic](functions/load_dynamic.md).

## Examples

```scrut
$ cp -R "$TESTDIR/fixtures/loading/." "$TMPDIR" && cd "$TMPDIR"
```

### Share code between files

<!-- fixture: fixtures/loading/lib/users.star -->
```python
load("starcm", "write")

ADMINS = ["ada", "grace"]

def greet(name):
    write("hello, %s" % name, label = "greet " + name)
```

<!-- fixture: fixtures/loading/main.star -->
```python
load("starcm", "write", say = "write")
load("lib/users.star", "ADMINS", "greet")

for admin in ADMINS:
    greet(admin)
say("aliases work too", label = "alias")
```

```scrut
$ starcm main.star
hello, ada
hello, grace
aliases work too
```

### Workspace-relative paths

<!-- fixture: fixtures/loading/workspace.star -->
```python
load("//lib/users.star", "greet")

greet("workspace")
```

`//lib/users.star` resolves against the directory starcm runs in, so this
works from `$TMPDIR` but not from `lib/`:

```scrut
$ starcm workspace.star
hello, workspace
```

```scrut
$ (cd lib && starcm ../workspace.star 2>&1 | tail -1)
Error: cannot load //lib/users.star: loading module ".*lib[/\\]+lib[/\\]+users\.star": open .*lib[/\\]lib[/\\]users\.star: (no such file or directory|The system cannot find the (file|path) specified\.) (regex)
```

## See also

- [examples/](../examples) and [tests/examples.md](../tests/examples.md) for nested loads and loads in loops.
