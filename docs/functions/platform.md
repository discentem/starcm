# platform

[← All functions](../README.md)

`platform` describes the machine starcm runs on. It returns a struct with two
fields, `os` and `arch`, named the way Go names them: `darwin`, `linux` or
`windows` for `os`, and `amd64`, `arm64` and so on for `arch`. Use it to pick
the right download or file name for each machine in one configuration.

Like [temp_dir](temp_dir.md) and [json](json.md), it's a plain helper: it takes
no `label` and returns a struct, not a `starcm_result`.

## Syntax

```python
load("starcm", "platform")

p = platform()
p.os      # "darwin"
p.arch    # "arm64"
```

## Arguments

None. `platform` takes no arguments.

## Returns

A struct with the fields:

| Field | Description |
| --- | --- |
| `os` | The operating system: `darwin`, `linux`, `windows`, and so on. |
| `arch` | The CPU architecture: `amd64`, `arm64`, and so on. |

## Examples

The examples run in a scratch copy of
[examples/platform/](../../examples/platform).

```scrut
$ cp -R "$TESTDIR/../../examples/platform/." "$TMPDIR" && cd "$TMPDIR"
```

### Build a file name for this machine

```python
# examples/platform/platform.star
load("starcm", "platform")

p = platform()
print("os=%s arch=%s" % (p.os, p.arch))
print("terraform_%s_%s.zip" % (p.os, p.arch))
```

The values depend on the machine, so the output is matched with a pattern:

```scrut
$ starcm platform.star
os=(darwin|linux|windows) arch=[a-z0-9]+ (regex)
terraform_(darwin|linux|windows)_[a-z0-9]+\.zip (regex)
```

## Errors and edge cases

These are here mostly as tests. They run in the copy of
[examples/platform/edge/](../../examples/platform/edge).

```scrut
$ cd "$TMPDIR/edge"
```

### Arguments are refused

```python
# examples/platform/edge/with_arg.star
load("starcm", "platform")

platform("darwin")
```

```scrut
$ starcm with_arg.star 2>&1
Traceback (most recent call last):
  with_arg.star:3:9: in <toplevel>
Error in platform: platform: takes no arguments
[1]
```

## See also

- [env](env.md) and [home_dir](home_dir.md), the other machine helpers.
- [temp_dir](temp_dir.md) for the system's temporary directory.
