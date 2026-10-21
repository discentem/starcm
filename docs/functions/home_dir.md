# home_dir

[← All functions](../README.md)

`home_dir` returns the user's home directory, optionally with more path parts
joined on. A leading `~` in a path expands to the same directory, so
`home_dir()` matches what [file](file.md) does with `~`: `$HOME` on macOS,
Linux and Windows, with the same fallbacks when it isn't set.

Like [temp_dir](temp_dir.md) and [json](json.md), it's a plain helper: it takes
no `label` and returns a string, not a `starcm_result`. It doesn't create
anything; pass `create_dirs = True` to [file](file.md) to create the folders.

## Syntax

```python
load("starcm", "home_dir")

home_dir()                           # e.g. "/Users/me" on macOS
home_dir(".config", "agent.toml")    # e.g. "/Users/me/.config/agent.toml"
```

## Arguments

| Argument | Type | Description |
| --- | --- | --- |
| `*parts` | strings | Path parts to join onto the home directory, with the platform's separator (`/`, or `\` on Windows). |

## Returns

The path as a string, with no trailing separator.

## Examples

The examples run in a scratch copy of [examples/home_dir/](../../examples/home_dir).
They point `HOME` at scrut's scratch directory, so they don't touch your real
home directory.

```scrut
$ cp -R "$TESTDIR/../../examples/home_dir/." "$TMPDIR" && cd "$TMPDIR"
```

### Write a config file under the home directory

```python
# examples/home_dir/home.star
load("starcm", "file", "home_dir")

path = home_dir(".config", "agent.toml")
r = file(label = "agent config", path = path, content = "debug = false\n", create_dirs = True)
print("changed=%s" % r.changed)
```

```scrut
$ HOME="$TMPDIR" USERPROFILE="$TMPDIR" starcm home.star && cat "$TMPDIR/.config/agent.toml"
changed=True
debug = false
```

## Errors and edge cases

These are here mostly as tests. They run in the copy of
[examples/home_dir/edge/](../../examples/home_dir/edge).

```scrut
$ cd "$TMPDIR/edge"
```

### Path parts must be strings

```python
# examples/home_dir/edge/bad_part.star
load("starcm", "home_dir")

home_dir(".config", 1)
```

```scrut
$ starcm bad_part.star 2>&1
Traceback (most recent call last):
  bad_part.star:3:9: in <toplevel>
Error in home_dir: home_dir: argument 2: got int, want string
[1]
```

## See also

- [file](file.md) with `create_dirs = True` to create folders under it.
- [platform](platform.md) and [env](env.md), the other machine helpers.
