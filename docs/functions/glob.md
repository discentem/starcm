# glob

[← All functions](../README.md)

Use `glob` to find the paths that match a pattern whose wildcards can appear
in any part of the path, directories included: for example every vendored Go
module, `third_party/*/go.mod`. To search a whole tree for files by name,
use [list_files](list_files.md) instead.

Like `list_files`, it reads the filesystem, so it takes a `label` and returns
a [starcm_result](../common-functionality.md#starcm_result); its `value` is
the list of paths.

## Syntax

```python
load("starcm", "glob", "here")

glob(here("third_party/*/go.mod"), label = "find modules").value
glob(pattern = "/etc/agent/*.d/*.conf", label = "find settings").value
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `pattern` | string | required | An absolute path, positionally or as `pattern =`, whose parts may hold the wildcards of Go's [filepath.Match](https://pkg.go.dev/path/filepath#Match): `*` (any run of characters), `?` (one character), `[abc]`, `[a-z]` and `[^abc]`. A leading `~` is expanded. A relative path is an error: build the pattern with [here](here.md), [home_dir](home_dir.md) or [temp_dir](temp_dir.md). An invalid pattern fails the run. |

Each wildcard matches within one part of the path: `*` never matches a `/`,
and there is no `**`. `*` and `?` also match a leading `.`, so `*` finds
hidden files. Outside Windows, `\` escapes a wildcard (`a\*b` matches only
`a*b`); on Windows `\` is a separator, like `/`.

`glob` also takes the [common arguments](../common-functionality.md#common-arguments):
`label` (required), `only_if`, `not_if`, `timeout` and
`what_if`. Only `only_if` and `not_if` affect it.

## Returns

A [starcm_result](../common-functionality.md#starcm_result) with `success =
True`, `changed = False` and `value` set to a sorted list of the absolute
paths that match, files and directories alike, with `/` separators on every
platform. It's `[]` when nothing matches, including when a directory in the
pattern doesn't exist. A pattern without wildcards returns that path if it
exists.

An error reading a directory the pattern looks in, such as permission
denied, stops the run rather than leaving its matches out.

## Examples

Each example runs in a scratch copy of
[examples/glob/](../../examples/glob), which has:

```scrut
$ cp -R "$TESTDIR/../../examples/glob/." "$TMPDIR" && cd "$TMPDIR"
```

```text
third_party/
├── a/go.mod
├── b/go.mod
└── c/README
```

### Match a wildcard in a directory

```python
# examples/glob/find.star
load("starcm", "glob", "here")

# * matches in any segment, so this looks one directory deep.
for m in glob(here("third_party/*/go.mod"), label = "find modules").value:
    print(m)

# Directories match too.
print(len(glob(here("third_party", "*"), label = "vendored dirs").value))
```

```scrut
$ starcm find.star
*/third_party/a/go.mod (glob)
*/third_party/b/go.mod (glob)
3
```

### Read every match

The paths are absolute, so they can go straight to another function:

```python
# examples/glob/modules.star
load("starcm", "glob", "here", "read_file")

for m in glob(here("third_party/*/go.mod"), label = "find modules").value:
    print(read_file(m, label = "read go.mod").value.splitlines()[0])
```

```scrut
$ starcm modules.star
module example.com/a
module example.com/b
```

## Errors and edge cases

These are here mostly as tests: CI runs them to check error messages and
behaviour that the examples above don't show. They run in the copy of
[examples/glob/edge/](../../examples/glob/edge).

```scrut
$ cd "$TMPDIR/edge"
```

### Nothing matches

A pattern that matches nothing returns `[]`, including when a directory in it
doesn't exist or is a file:

```python
# examples/glob/edge/no_match.star
load("starcm", "glob", "here")

print(glob(here("../third_party/*/Cargo.toml"), label = "no matches").value)
print(glob(here("../missing/*/go.mod"), label = "missing dir").value)
print(glob(here("../third_party/c/README/*"), label = "file, not dir").value)
```

```scrut
$ starcm no_match.star
[]
[]
[]
```

### ~ is expanded

```python
# examples/glob/edge/tilde.star
load("starcm", "glob")

for m in glob("~/third_party/*/go.mod", label = "under home").value:
    print(m)
```

```scrut
$ HOME="$TMPDIR" USERPROFILE="$TMPDIR" starcm tilde.star
*/third_party/a/go.mod (glob)
*/third_party/b/go.mod (glob)
```

### A relative pattern stops the run

```python
# examples/glob/edge/relative.star
load("starcm", "glob")

glob("third_party/*/go.mod", label = "relative")
```

```scrut
$ starcm relative.star 2>&1
Traceback (most recent call last):
  relative.star:3:5: in <toplevel>
Error in glob: glob(label="relative"): pattern: "third_party/*/go.mod" is not an absolute path: write the full path, or pass the result of the step that made the file
[1]
```

### An invalid pattern stops the run

```python
# examples/glob/edge/bad_pattern.star
load("starcm", "glob", "here")

glob(here("../third_party/*/[.mod"), label = "bad pattern")
```

```scrut
$ starcm bad_pattern.star 2>&1
Traceback (most recent call last):
  bad_pattern.star:3:5: in <toplevel>
Error in glob: glob(label="bad pattern"): invalid pattern "*third_party*[.mod": syntax error in pattern (glob)
[1]
```

## See also

- [list_files](list_files.md) to search a directory tree for files by name.
- [path.exists](path.md) to check a single path.
