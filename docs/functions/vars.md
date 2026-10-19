# vars

[← All functions](../README.md)

`vars` is a dict of the values passed on the command line with
`--var name=value`. Use it to run the same configuration in different
environments: a base URL, a release channel, a platform. It's a value, not a
function: it takes no `label` and returns nothing.

## Syntax

```python
load("starcm", "vars")

vars["base_url"]              # fails the run if --var base_url=... wasn't passed
vars.get("channel", "stable") # with a default
```

```bash
starcm --var base_url=https://config.example.com --var channel=beta main.star
```

## Details

- Keys and values are strings. Convert them yourself, e.g. `int(vars["port"])`
  or `vars.get("debug") == "true"`.
- `--var` is repeatable. Values are kept whole, commas included, and
  `--var name=` sets an empty string. The last `--var` for a name wins.
- `vars` is frozen and shared by every file in the run, including files
  loaded with `load()` and [load_dynamic](load_dynamic.md).
- A `--var` without `=` is rejected before anything runs.

## Examples

Each example runs in a scratch copy of
[examples/vars/](../../examples/vars).

```scrut
$ cp -R "$TESTDIR/../../examples/vars/." "$TMPDIR" && cd "$TMPDIR"
```

```python
# examples/vars/vars.star
load("starcm", "vars")

print(vars)
print("channel=%s" % vars.get("channel", "stable"))
```

### Without `--var`

```scrut
$ starcm vars.star
{}
channel=stable
```

### With `--var`

```scrut
$ starcm --var channel=beta --var hosts=a,b vars.star
{"channel": "beta", "hosts": "a,b"}
channel=beta
```

### Malformed `--var`

```scrut
$ starcm --var channel vars.star 2>&1
--var "channel": want name=value
[1]
```

## Errors and edge cases

These are here mostly as tests: CI runs them to check error messages and
behaviour that the examples above don't show. They run in the copy of
[examples/vars/edge/](../../examples/vars/edge).

```scrut
$ cd "$TMPDIR/edge"
```

### Values are kept whole: commas, URLs and empty values

```python
# examples/vars/edge/vars.star
load("starcm", "vars")

print(vars)
print("base=%s" % vars.get("base", "unset"))
```

```scrut
$ starcm --var base=http://localhost:8000/stable --var list=a,b --var empty= vars.star
{"base": "http://localhost:8000/stable", "empty": "", "list": "a,b"}
base=http://localhost:8000/stable
```

## See also

- [Command line](../cli.md).
- [download](download.md) and [load_dynamic](load_dynamic.md) have examples that use `vars`.
