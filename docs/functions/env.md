# env

[← All functions](../README.md)

`env` returns the value of an environment variable as a string, or a default
when the variable isn't set. Use it to read settings the machine provides,
such as a region or a token that a CI system exports, without failing when
it's missing.

Like [temp_dir](temp_dir.md) and [json](json.md), it's a plain helper: it takes
no `label` and returns a string, not a `starcm_result`.

## Syntax

```python
load("starcm", "env")

env("REGION")                        # None if REGION isn't set
env("REGION", default = "us-east-1") # "us-east-1" if REGION isn't set
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `name` | string | required | The environment variable's name. Names are case-sensitive on macOS and Linux, and not on Windows. |
| `default` | string | `None` | Returned when the variable isn't set. |

A variable that is set to an empty string returns `""`, not `default`.

## Returns

The variable's value as a string. When the variable isn't set, `default`, or
`None` if no `default` was given.

## Examples

The examples run in a scratch copy of [examples/env/](../../examples/env).

```scrut
$ cp -R "$TESTDIR/../../examples/env/." "$TMPDIR" && cd "$TMPDIR"
```

### Read a setting with a fallback

```python
# examples/env/env.star
load("starcm", "env")

print("region=%s" % env("STARCM_EXAMPLE_REGION", default = "us-east-1"))
print("unset=%s" % env("STARCM_EXAMPLE_UNSET"))
print("empty=[%s]" % env("STARCM_EXAMPLE_EMPTY", default = "fallback"))
```

The variables aren't set in the first run, so `env` returns the default, or
`None`. The third one is set to nothing, so it returns an empty string:

```scrut
$ STARCM_EXAMPLE_EMPTY= starcm env.star
region=us-east-1
unset=None
empty=[]
```

Set the variable and `env` returns its value:

```scrut
$ STARCM_EXAMPLE_REGION=eu-west-1 starcm env.star | head -1
region=eu-west-1
```

## Errors and edge cases

These are here mostly as tests. They run in the copy of
[examples/env/edge/](../../examples/env/edge).

```scrut
$ cd "$TMPDIR/edge"
```

### The name is required

```python
# examples/env/edge/no_name.star
load("starcm", "env")

env()
```

```scrut
$ starcm no_name.star 2>&1
Traceback (most recent call last):
  no_name.star:3:4: in <toplevel>
Error in env: env: missing argument for name
[1]
```

### The default must be a string

```python
# examples/env/edge/bad_default.star
load("starcm", "env")

env("STARCM_EXAMPLE_REGION", default = 1)
```

```scrut
$ starcm bad_default.star 2>&1
Traceback (most recent call last):
  bad_default.star:3:4: in <toplevel>
Error in env: env: default: got int, want string or None
[1]
```

## See also

- [platform](platform.md) and [home_dir](home_dir.md), the other machine helpers.
