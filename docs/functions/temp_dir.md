# temp_dir

[← All functions](../README.md)

`temp_dir` returns the system's temporary directory, optionally with more
path parts joined on: `$TMPDIR` (or `/tmp`) on macOS and Linux, and `%TMP%`
or `%TEMP%` on Windows. Use it for scratch files that shouldn't land in the
directory starcm runs in. The directory is the same on every run, so
[file](file.md) and [download](download.md) stay idempotent there.

Like [json](json.md) and [vars](vars.md), it's a plain helper: it takes no
`label` and returns a string, not a `starcm_result`. It doesn't create
anything; pass `create_dirs = True` to [file](file.md), for example, to
create the folders.

## Syntax

```python
load("starcm", "temp_dir")

temp_dir()                            # e.g. "/var/folders/xy/abc123/T" on macOS
temp_dir("starcm", "cache", "a.json") # ".../T/starcm/cache/a.json"
```

## Arguments

| Argument | Type | Description |
| --- | --- | --- |
| `*parts` | strings | Path parts to join onto the temporary directory, with the platform's separator (`/`, or `\` on Windows). |

## Returns

The path as a string, with no trailing separator.

## Examples

The examples run in a scratch copy of
[examples/temp_dir/](../../examples/temp_dir), with the temporary directory
pointed at scrut's scratch directory.

```scrut
$ cp -R "$TESTDIR/../../examples/temp_dir/." "$TMPDIR" && cd "$TMPDIR" && export TMP="$TMPDIR" TEMP="$TMPDIR"
```

### Keep a scratch file in the temporary directory

```python
# examples/temp_dir/scratch.star
load("starcm", "file", "read_file", "temp_dir")

print(temp_dir())
print(temp_dir("starcm", "cache", "agent.json"))

note = temp_dir("starcm-docs", "note.txt")
r = file(label = "scratch note", path = note, content = "kept between runs\n", create_dirs = True)
print("changed=%s" % r.changed)
print(read_file(note).strip())
```

```scrut
$ starcm scratch.star
* (glob)
*starcm?cache?agent.json (glob)
changed=True
kept between runs
```

The second run finds the note where the first left it:

```scrut
$ starcm scratch.star | tail -2
changed=False
kept between runs
```

## See also

- [file](file.md) with `create_dirs = True` to create folders under it.
