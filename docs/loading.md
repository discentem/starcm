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
| `starcm` | [download](functions/download.md), [s3_download](functions/s3_download.md), [exec](functions/exec.md), [file](functions/file.md), [template](functions/template.md), [install_mac_pkg](functions/install_mac_pkg.md), [install_remote_mac_pkg](functions/install_remote_mac_pkg.md), [write](functions/write.md), [date](functions/date.md), [load_dynamic](functions/load_dynamic.md), [load_remote](functions/load_remote.md), [read_file](functions/read_file.md), [sha256](functions/sha256.md), [inspect](functions/inspect.md), [sign](functions/sign.md), [verify_all](functions/verify_all.md), [json](functions/json.md), [path](functions/path.md), [vars](functions/vars.md), [temp_dir](functions/temp_dir.md), [dedent](functions/dedent.md) |
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
[load_dynamic](functions/load_dynamic.md); to fetch one from a server or S3
bucket, use [load_remote](functions/load_remote.md).

## Examples

```scrut
$ cp -R "$TESTDIR/../examples/loading/." "$TMPDIR" && cd "$TMPDIR"
```

### Share code between files

```python
# examples/loading/lib/users.star
load("starcm", "write")

ADMINS = ["ada", "grace"]

def greet(name):
    write("hello, %s" % name, label = "greet " + name)
```

```python
# examples/loading/main.star
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

```python
# examples/loading/workspace.star
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

### `if` and `for` at the top level

```python
# examples/loading/if_statements.star
load("starcm", "exec")
a = exec(
    label               = "explicitly exit 2",
    cmd                = "sh", 
    args               = ["-c", "echo 'we expect to exit 2'; exit 2"],
    expected_exit_code = 2,
)
if a.success == True:
    print("party!")
else:
    print("no party :(")
```

```scrut
$ starcm if_statements.star
party!
```

### Load files in a loop

```python
# examples/loading/loop.star
load("starcm", "load_dynamic")

for f in ['nested_files/e.star', 'nested_files/subfolder/subfolder2/subfolder3/d.star']:
    load_dynamic(f, label=f)
```

```scrut
$ starcm loop.star
hello from e.star
hello from d.star
```

### A chain of nested files

`b.star` loads `c.star` by a path relative to itself, which loads `d.star`
the same way and `e.star` by a workspace-relative path:

```python
# examples/loading/nested_files/subfolder/b.star
e = ""

print('hello from b.star')

load("starcm", "load_dynamic")
load_dynamic("subfolder2/c.star", label="load c.star from b.star")
```

```python
# examples/loading/nested_files/subfolder/subfolder2/c.star
print("hello from c.star")

load("starcm", "load_dynamic")

load_dynamic("subfolder3/d.star", label="load d.star from c.star")

load_dynamic("//examples/loading/nested_files/e.star", label="load e.star from c.star")
```

`//examples/loading/nested_files/e.star` resolves against the directory
starcm runs in, so this runs from a directory laid out like the repository:

```scrut
$ mkdir -p repo/examples/loading && cp -R nested_files repo/examples/loading/ && (cd repo && starcm examples/loading/nested_files/subfolder/b.star)
hello from b.star
hello from c.star
hello from d.star
hello from e.star
```

## See also

- [examples/loading/](../examples/loading), the files these examples run.
