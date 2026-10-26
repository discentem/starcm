# path

[← All functions](../README.md)

`path` is a module of path functions. Load it from `starcm`, like
[json](json.md):

- `path.exists` checks whether a file or directory is there.
- `path.join` builds a path from parts.
- `path.glob` finds the paths that match a pattern with wildcards in any
  part, directories included, such as `third_party/*/go.mod`.
- `path.list` finds the files under a directory, recursively, by name.

They only read, so they're plain functions: they take no `label`, return
their answer directly rather than a `starcm_result`, and work in `if`,
`for` and comprehensions. Any error stops the run.

## Syntax

```python
load("starcm", "here", "path")

path.exists("/usr/local/bin/go")                # True or False
path.join("/opt/agent", "bin", "agent")         # "/opt/agent/bin/agent"
path.glob(here("third_party/*/go.mod"))         # ["/.../third_party/a/go.mod", ...]
path.list(here("config"), pattern = "*.star")   # ["lib/macos.star", "main.star", ...]
```

## Functions

| Function | Returns | Description |
| --- | --- | --- |
| `path.exists(path)` | bool | Whether `path` is a file or directory, following symlinks (a link to nothing doesn't exist). |
| `path.join(*parts)` | string | The string `parts` joined with the platform's separator (`/`, or `\` on Windows) and cleaned, as Go's [filepath.Join](https://pkg.go.dev/path/filepath#Join) does: repeated separators, `.` and `..` are resolved and empty parts are ignored. It takes at least one part, doesn't touch the filesystem and doesn't expand `~`. |
| `path.glob(pattern)` | list of strings | The absolute paths that match `pattern`, files and directories alike, sorted, with `/` separators on every platform. |
| `path.list(dir, pattern = "*")` | list of strings | The files under `dir`, recursively, whose name (not path) matches `pattern`, sorted, relative to `dir`, with `/` separators on every platform. Directories aren't listed. |

`path`, `pattern` (for `path.glob`) and `dir` must be absolute, and a
leading `~` is expanded: build them with [here](here.md),
[home_dir](home_dir.md) or [temp_dir](temp_dir.md). Arguments can be passed
positionally or by name.

### path.exists

`path.exists` returns `False` only when the path doesn't exist, including
when a part of it that should be a directory is a file. Any other error,
such as permission denied on a directory above it, stops the run: a path
starcm can't check never looks missing.

### path.glob

Each part of the pattern may hold the wildcards of Go's
[filepath.Match](https://pkg.go.dev/path/filepath#Match): `*` (any run of
characters), `?` (one character), `[abc]`, `[a-z]` and `[^abc]`. Each
wildcard matches within one part of the path: `*` never matches a `/`, and
there is no `**`. `*` and `?` also match a leading `.`, so `*` finds hidden
files. Outside Windows, `\` escapes a wildcard (`a\*b` matches only `a*b`);
on Windows `\` is a separator, like `/`.

It returns `[]` when nothing matches, including when a directory in the
pattern doesn't exist. A pattern without wildcards returns that path if it
exists. An invalid pattern stops the run, and so does an error reading a
directory the pattern looks in, such as permission denied, rather than
leaving its matches out.

### path.list

`pattern` is matched against each file's name with the same wildcards. Use
`path.list` to search a whole tree by name, and `path.glob` when the
wildcards are in directory names.

## Examples

Each example runs in a scratch copy of
[examples/path/](../../examples/path), which has:

```scrut
$ cp -R "$TESTDIR/../../examples/path/." "$TMPDIR" && cd "$TMPDIR"
```

```text
files/
└── agent.conf
glob/third_party/
├── a/go.mod
├── b/go.mod
└── c/README
list/config/
├── lib/
│   ├── macos.star
│   └── windows.star
├── main.star
└── notes.txt
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

### Match a wildcard in a directory

```python
# examples/path/glob/find.star
load("starcm", "here", "path")

# * matches in any segment, so this looks one directory deep.
for m in path.glob(here("third_party/*/go.mod")):
    print(m)

# Directories match too.
print(len(path.glob(here("third_party", "*"))))
```

```scrut
$ cd glob && starcm find.star
*/third_party/a/go.mod (glob)
*/third_party/b/go.mod (glob)
3
```

The paths are absolute, so they can go straight to another function:

```python
# examples/path/glob/modules.star
load("starcm", "here", "path", "read_file")

for m in path.glob(here("third_party/*/go.mod")):
    print(read_file(m, label = "read go.mod").value.splitlines()[0])
```

```scrut
$ starcm modules.star && cd ..
module example.com/a
module example.com/b
```

### List files by name

```python
# examples/path/list/list.star
load("starcm", "here", "path")

print(path.list(here("config")))
print(path.list(here("config"), pattern = "*.star"))
print(path.list(dir = here("config/lib"), pattern = "w*"))
```

```scrut
$ cd list && starcm list.star
["lib/macos.star", "lib/windows.star", "main.star", "notes.txt"]
["lib/macos.star", "lib/windows.star", "main.star"]
["windows.star"]
```

### Build a manifest of hashes

```python
# examples/path/list/manifest.star
load("starcm", "here", "path", "sha256")

for f in path.list(here("config"), pattern = "*.star"):
    h = sha256(here("config", f), label = "hash " + f).value
    print("%s  %s" % (h[:12], f))
```

```scrut
$ starcm manifest.star && cd ..
cbdeb467b4cb  lib/macos.star
76708f9814ef  lib/windows.star
2377a494b815  main.star
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
Error in path.exists: path.exists: path: "files/agent.conf" is not an absolute path: write the full path, or pass the result of the step that made the file
[1]
```

### Nothing matches

`path.glob` returns `[]` when nothing matches, including when a directory in
the pattern doesn't exist or is a file:

```python
# examples/path/edge/glob_no_match.star
load("starcm", "here", "path")

print(path.glob(here("../glob/third_party/*/Cargo.toml")))   # nothing matches
print(path.glob(here("../glob/missing/*/go.mod")))           # missing directory
print(path.glob(here("../glob/third_party/c/README/*")))     # a file, not a directory
```

```scrut
$ starcm glob_no_match.star
[]
[]
[]
```

### ~ is expanded

```python
# examples/path/edge/glob_tilde.star
load("starcm", "path")

for m in path.glob("~/glob/third_party/*/go.mod"):
    print(m)
```

```scrut
$ HOME="$TMPDIR" USERPROFILE="$TMPDIR" starcm glob_tilde.star
*/glob/third_party/a/go.mod (glob)
*/glob/third_party/b/go.mod (glob)
```

### A relative pattern stops the run

```python
# examples/path/edge/glob_relative.star
load("starcm", "path")

path.glob("third_party/*/go.mod")
```

```scrut
$ starcm glob_relative.star 2>&1
Traceback (most recent call last):
  glob_relative.star:3:10: in <toplevel>
Error in path.glob: path.glob: pattern: "third_party/*/go.mod" is not an absolute path: write the full path, or pass the result of the step that made the file
[1]
```

### An invalid pattern stops the run

```python
# examples/path/edge/glob_bad_pattern.star
load("starcm", "here", "path")

path.glob(here("../glob/third_party/*/[.mod"))
```

```scrut
$ starcm glob_bad_pattern.star 2>&1
Traceback (most recent call last):
  glob_bad_pattern.star:3:10: in <toplevel>
Error in path.glob: path.glob: invalid pattern "*third_party*[.mod": syntax error in pattern (glob)
[1]
```

```python
# examples/path/edge/list_bad_pattern.star
load("starcm", "here", "path")

path.list(here("../list/config"), pattern = "[")
```

```scrut
$ starcm list_bad_pattern.star 2>&1
Traceback (most recent call last):
  list_bad_pattern.star:3:10: in <toplevel>
Error in path.list: path.list: invalid pattern "[": syntax error in pattern
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
- [sign](sign.md) and [inspect](inspect.md) have examples that go through every
  `.star` file `path.list` finds.
- [Common functionality](../common-functionality.md) for `only_if` and `not_if`.
