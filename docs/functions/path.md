# path

[← All functions](../README.md)

`path` is a module of path helpers: `path.exists` checks whether a file or
directory is there, and `path.join` builds a path from parts. Load it from
`starcm`, like [json](json.md). Like [here](here.md) and
[temp_dir](temp_dir.md), its functions are plain helpers: they take no
`label` and return plain values, not a `starcm_result`, so they work
directly in `if`, `only_if` and `not_if`.

## Syntax

```python
load("starcm", "path")

path.exists("/usr/local/bin/go")          # True or False
path.join("/opt/agent", "bin", "agent")   # "/opt/agent/bin/agent"
```

## Functions

| Function | Returns | Description |
| --- | --- | --- |
| `path.exists(p)` | bool | Whether `p` is a file or directory, following symlinks (a link to nothing doesn't exist). `p` must be absolute; a leading `~` is expanded. |
| `path.join(*parts)` | string | The string `parts` joined with the platform's separator (`/`, or `\` on Windows) and cleaned, as Go's [filepath.Join](https://pkg.go.dev/path/filepath#Join) does: repeated separators, `.` and `..` are resolved and empty parts are ignored. It takes at least one part, doesn't touch the filesystem and doesn't expand `~`. |

`path.exists` returns `False` only when `p` doesn't exist, including when a
part of it that should be a directory is a file. Any other error, such as
permission denied on a directory above `p`, stops the run: a path starcm
can't check never looks missing.

## Examples

Each example runs in a scratch copy of
[examples/path/](../../examples/path), which has `files/agent.conf`.

```scrut
$ cp -R "$TESTDIR/../../examples/path/." "$TMPDIR" && cd "$TMPDIR"
```

### Check whether a path exists

```python
# examples/path/exists.star
load("starcm", "here", "path")

print(path.exists(here("files/agent.conf")))     # a file
print(path.exists(here("files")))                # a directory
print(path.exists(here("files/missing.conf")))   # nothing there
```

```scrut
$ starcm exists.star
True
True
False
```

### Do something only the first time

This example points `HOME` at scrut's scratch directory, so it doesn't touch
your real home directory.

```python
# examples/path/first_run.star
load("starcm", "file", "home_dir", "path")

marker = home_dir(".agent", "installed")
if path.exists(marker):
    print("already installed")
else:
    file(label = "mark installed", path = marker, content = "", create_dirs = True)
    print("first run: installed")
```

```scrut
$ HOME="$TMPDIR" USERPROFILE="$TMPDIR" starcm first_run.star && HOME="$TMPDIR" USERPROFILE="$TMPDIR" starcm first_run.star
first run: installed
already installed
```

### Join path parts

```python
# examples/path/join.star
load("starcm", "path")

print(path.join("/opt/agent", "bin", "agent"))
print(path.join("/opt/agent/", "../tools", ""))   # cleaned, like Go's filepath.Join
```

On Windows the separator is `\`:

```scrut
$ starcm join.star
[/\\]opt[/\\]agent[/\\]bin[/\\]agent (regex)
[/\\]opt[/\\]tools (regex)
```

## Errors and edge cases

These are here mostly as tests: CI runs them to check error messages and
behaviour that the examples above don't show. They run in the copy of
[examples/path/edge/](../../examples/path/edge).

```scrut
$ cd "$TMPDIR/edge"
```

### A file where a directory should be

```python
# examples/path/edge/not_dir.star
load("starcm", "here", "path")

# A file where a directory should be: the path doesn't exist.
print(path.exists(here("../files/agent.conf/x")))
```

```scrut
$ starcm not_dir.star
False
```

### path.exists needs an absolute path

```python
# examples/path/edge/relative.star
load("starcm", "path")

path.exists("files/agent.conf")
```

```scrut
$ starcm relative.star 2>&1
Traceback (most recent call last):
  relative.star:3:12: in <toplevel>
Error in path.exists: path.exists: "files/agent.conf" is not an absolute path: write the full path, or pass the result of the step that made the file
[1]
```

### path.join needs at least one string

```python
# examples/path/edge/join_no_parts.star
load("starcm", "path")

path.join()
```

```scrut
$ starcm join_no_parts.star 2>&1
Traceback (most recent call last):
  join_no_parts.star:3:10: in <toplevel>
Error in path.join: path.join: got no arguments, want at least one
[1]
```

```python
# examples/path/edge/join_bad_part.star
load("starcm", "path")

path.join("/opt", 1)
```

```scrut
$ starcm join_bad_part.star 2>&1
Traceback (most recent call last):
  join_bad_part.star:3:10: in <toplevel>
Error in path.join: path.join: argument 2: got int, want string
[1]
```

## See also

- [here](here.md), [home_dir](home_dir.md) and [temp_dir](temp_dir.md), which
  also join parts onto a directory.
- [glob](glob.md) to find every path that matches a pattern.
- [Common functionality](../common-functionality.md) for `only_if` and `not_if`.
