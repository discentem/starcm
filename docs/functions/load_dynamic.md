# load_dynamic

[← All functions](../README.md)

Use `load_dynamic` to run another `.star` file and get its globals, when the
path isn't known until the configuration runs. Starlark's `load()` statement
needs a literal path and must come at the top of a file; `load_dynamic` is an
ordinary function call, so the path can come from [vars](vars.md), a loop or
another function's result, and the call can be skipped with `only_if`.

A module is loaded once per run: loading the same path again returns the same
globals without running the file again. Under
[--trusted-keys](../cli.md#--trusted-keys-dir), the loaded file needs a
valid signature, like any other `.star` file.

## Syntax

```python
load("starcm", "load_dynamic")

load_dynamic("platforms/macos.star", label = "load macOS settings")
load_dynamic(module = "platforms/macos.star", label = "load macOS settings")
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `module` | string | required | Path of the `.star` file, positionally or as `module =`. Paths starting with `//` are relative to the directory starcm runs in; other relative paths are relative to the calling file. See [Loading files](../loading.md). |

`load_dynamic` also takes the [common arguments](../common-functionality.md#common-arguments):
`label` (required), `only_if`, `not_if`, `timeout` and
`what_if`. Only `only_if` and `not_if` affect it.

## Returns

A [starcm_result](../common-functionality.md#starcm_result):

| Field | Value |
| --- | --- |
| `success` | `True` |
| `changed` | `False` |
| `message` | `module "<path>" loaded successfully` |
| `value` | A frozen dict of the module's globals: `value["NAME"]` |

A missing file, a syntax error or an error while the module runs stops the
run.

## Examples

Each example runs in a scratch copy of
[examples/load_dynamic/](../../examples/load_dynamic), which has a settings
file per platform:

```scrut
$ cp -R "$TESTDIR/../../examples/load_dynamic/." "$TMPDIR" && cd "$TMPDIR"
```

```python
# examples/load_dynamic/platforms/macos.star
load("starcm", "write")

SCREEN_LOCK_SECONDS = 300

def configure():
    write("macOS: require a password %d seconds after the screen saver starts" % SCREEN_LOCK_SECONDS, label = "macOS screen lock")
```

```python
# examples/load_dynamic/platforms/windows.star
load("starcm", "write")

SCREEN_LOCK_SECONDS = 600

def configure():
    write("Windows: lock the screen after %d seconds idle" % SCREEN_LOCK_SECONDS, label = "Windows screen lock")
```

### Pick a module with `--var`

```python
# examples/load_dynamic/by_platform.star
load("starcm", "load_dynamic", "vars")

platform = vars.get("platform", "macos")
r = load_dynamic("platforms/%s.star" % platform, label = "load %s settings" % platform)
print("success=%s changed=%s" % (r.success, r.changed))
print(sorted(r.value.keys()))
r.value["configure"]()
```

```scrut
$ starcm by_platform.star
success=True changed=False
["SCREEN_LOCK_SECONDS", "configure"]
macOS: require a password 300 seconds after the screen saver starts
```

```scrut
$ starcm --var platform=windows by_platform.star
success=True changed=False
["SCREEN_LOCK_SECONDS", "configure"]
Windows: lock the screen after 600 seconds idle
```

### Load modules in a loop

```python
# examples/load_dynamic/all_platforms.star
load("starcm", "load_dynamic")

for platform in ["macos", "windows"]:
    settings = load_dynamic(module = "platforms/%s.star" % platform, label = "load " + platform).value
    print("%s locks the screen after %d seconds" % (platform, settings["SCREEN_LOCK_SECONDS"]))
```

```scrut
$ starcm all_platforms.star
macos locks the screen after 300 seconds
windows locks the screen after 600 seconds
```

### Load a module only when it's needed

A skipped call returns `value = None`.

```python
# examples/load_dynamic/optional.star
load("starcm", "load_dynamic", "vars")

r = load_dynamic(
    "extras/developer_tools.star",
    label   = "developer tools",
    only_if = vars.get("developer") == "yes",
)
if r.value:
    r.value["configure"]()
else:
    print("developer tools skipped")
```

```scrut {output_stream: combined}
$ starcm optional.star
INFO: [developer tools]: skipping load_dynamic(label="developer tools") because only_if was false
developer tools skipped
```

```scrut
$ starcm --var developer=yes optional.star
installing developer tools
```

## Errors and edge cases

These are here mostly as tests: CI runs them to check error messages and
behaviour that the examples above don't show. They run in the copy of
[examples/load_dynamic/edge/](../../examples/load_dynamic/edge).

```scrut
$ cd "$TMPDIR/edge"
```

### Missing module fails the run

```python
# examples/load_dynamic/edge/missing.star
load("starcm", "load_dynamic")

load_dynamic("nope.star", label = "missing module")
```

```scrut
$ starcm missing.star 2>&1
Traceback (most recent call last):
  missing.star:3:13: in <toplevel>
Error in load_dynamic: loading module ".*[/\\]nope\.star": open .*[/\\]nope\.star: (no such file or directory|The system cannot find the file specified\.) (regex)
[1]
```

## See also

- [Loading files](../loading.md) for `load()` and how paths resolve.
- [load_remote](load_remote.md) to fetch a `.star` file from a server or S3 bucket and load it.
- [inspect](inspect.md) reports each `load_dynamic` call's path, or `null` when it's computed.
