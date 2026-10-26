# inspect

[← All functions](../README.md)

Use `inspect` to find out what a `.star` file loads and calls **without
running it**, so you can enforce rules such as "this file may only use
`download`" before it's signed or deployed. It's available as the
`inspect()` function and as the `starcm inspect` command, which prints the
same report as JSON.

Calls are resolved through `load()` aliases, so
`load("starcm", run = "exec"); run(...)` is reported as a call to `exec`.
Each call also says which loaded values reach its arguments, so a policy can
check, for example, that no `--var` value ends up in an `exec` command.

## Syntax

```python
load("starcm", "here", "inspect")

inspect(here("bootstrap.star"), label = "check bootstrap").value
inspect(path = here("bootstrap.star"), label = "check bootstrap").value
```

```bash
starcm inspect FILE...
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `path` | string | required | The `.star` file to inspect, positionally or as `path =`. An absolute path; a leading `~` expands to the home directory. `starcm inspect` takes paths relative to the directory it runs in. |

`inspect` also takes the [common arguments](../common-functionality.md#common-arguments):
`label` (required), `only_if`, `not_if`, `timeout` and
`what_if`. Only `only_if` and `not_if` affect it.

## Returns

A [starcm_result](../common-functionality.md#starcm_result) with `success =
True`, `changed = False` and `value` set to a report dict. `starcm inspect`
prints a JSON list with one report per file.

| Key | Description |
| --- | --- |
| `file` | The path as given. |
| `loads` | One entry per `load()` statement: `module`, `line`, and `names`, which maps each local name to the name it was loaded as. |
| `calls` | Direct calls to loaded names: `module`, `function` (the original name, after aliases) and `line`. Calls to `load_dynamic` also have `path`: its literal module path, or `None` / `null` when the path is computed at runtime. Calls to [load_remote](load_remote.md) and [install_remote_mac_pkg](install_remote_mac_pkg.md) have `url` the same way. |
| `calls[].inputs` | Only on calls with arguments that come from loaded names. One entry per such argument: `arg` (the keyword, the position `"0"`, `"1"`, ... of a positional argument, or `"*"` / `"**"`) and `uses`, the loaded names its value is computed from, as `module` and `function`. See [What reaches a call's arguments](#what-reaches-a-calls-arguments). |
| `values` | Other uses of loaded names, such as `vars["base_url"]` or passing a function around: `module`, `function` and `line`. Calls made through such values can't be known statically. |

A file that can't be read or parsed stops the run (or makes `starcm inspect`
exit 1).

## Examples

Each example runs in a scratch copy of
[examples/inspect/](../../examples/inspect).

```scrut
$ cp -R "$TESTDIR/../../examples/inspect/." "$TMPDIR" && cd "$TMPDIR"
```

### Report from the command line

```python
# examples/inspect/sneaky.star
load("starcm", run = "exec")

run(label = "innocent", cmd = "sh", args = ["-c", "echo pwned"])
```

The alias doesn't hide the `exec` call, and nothing runs:

```scrut
$ starcm inspect sneaky.star
[
  {
    "file": "sneaky.star",
    "loads": [
      {
        "module": "starcm",
        "line": 1,
        "names": {
          "run": "exec"
        }
      }
    ],
    "calls": [
      {
        "module": "starcm",
        "function": "exec",
        "line": 3
      }
    ],
    "values": []
  }
]
```

### Find computed `load_dynamic` paths

`bootstrap.star` downloads an agent, loads settings that apply on every
platform from a fixed path, and loads the settings for the platform it runs
on from a path built from `--var platform`:

```python
# examples/inspect/bootstrap.star
load("starcm", "download", "load_dynamic", "vars")

download(
    label   = "fetch agent",
    url     = vars["base_url"] + "/agent.pkg",
    sha256  = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
)
load_dynamic("settings/common.star", label = "settings for every platform")
load_dynamic("settings/%s.star" % vars["platform"], label = "settings for this platform")
```

With `jq`, list each call. `inspect` reports the fixed path, and `null` for
the one that's only known when the file runs. `inputs` shows that the
download's `url` and the second path come from `vars`:

```scrut
$ starcm inspect bootstrap.star | jq -c '.[].calls[]'
{"module":"starcm","function":"download","line":3,"inputs":[{"arg":"url","uses":[{"module":"starcm","function":"vars"}]}]}
{"module":"starcm","function":"load_dynamic","line":8,"path":"settings/common.star"}
{"module":"starcm","function":"load_dynamic","line":9,"path":null,"inputs":[{"arg":"0","uses":[{"module":"starcm","function":"vars"}]}]}
```

```scrut
$ starcm inspect bootstrap.star | jq -c '[.[].values[] | [.function, .line]]'
[["vars",5],["vars",9]]
```

### Enforce a policy in Starlark

```python
# examples/inspect/policy.star
load("starcm", "here", "inspect", "path")

ALLOWED = ["download", "here", "load_dynamic"]

for f in path.list(here(), pattern = "*.star"):
    # Only this folder's configuration files, not this script or subfolders.
    if f == "policy.star" or "/" in f:
        continue
    report = inspect(here(f), label = "inspect " + f).value
    bad = [c["function"] for c in report["calls"] if c["function"] not in ALLOWED]
    print("%s: %s" % (f, "ok" if not bad else "uses %s" % ", ".join(bad)))
```

```scrut
$ starcm policy.star
bootstrap.star: ok
sneaky.star: uses exec
```

### Keep `--var` values out of commands

A `--var` value is whatever the command line that runs starcm says. A policy
can refuse configuration that passes one to `exec`. `computer_name.star`
builds a name from two `--var` values before passing it on:

```python
# examples/inspect/lint/computer_name.star
load("starcm", "exec", "vars")

name = "%s-%s" % (vars["team"], vars["serial"])
exec(label = "set computer name", cmd = "scutil", args = ["--set", "ComputerName", name])
```

```python
# examples/inspect/lint/show_name.star
load("starcm", "exec")

exec(label = "show computer name", cmd = "scutil", args = ["--get", "ComputerName"])
```

```python
# examples/inspect/lint/no_vars_in_exec.star
load("starcm", "here", "inspect", "path")

for f in path.list(here(), pattern = "*.star"):
    for c in inspect(here(f), label = "inspect " + f).value["calls"]:
        if c["function"] != "exec":
            continue
        for i in c.get("inputs", []):
            if "vars" in [u["function"] for u in i["uses"]]:
                print("%s:%d: exec's %s argument comes from vars" % (f, c["line"], i["arg"]))
```

```scrut
$ cd lint && starcm no_vars_in_exec.star
computer_name.star:4: exec's args argument comes from vars
```

```scrut
$ starcm inspect computer_name.star | jq -c '.[0].calls[0].inputs'
[{"arg":"args","uses":[{"module":"starcm","function":"vars"}]}]
```

### What reaches a call's arguments

`inputs` follows a value through local variables, `+=`, `d[k] = v`, list
and dict methods such as `append` and `update`, `for` loops, comprehensions,
and the parameters and return values of functions defined in the same file.
It goes by name and ignores the order of statements, so it can report a use
that can't happen but shouldn't miss one that can. It doesn't follow values
into or out of other files: a function loaded from another file is reported
as a use, like `read_file` in `read_file(...).value`.

## Errors and edge cases

These are here mostly as tests: CI runs them to check error messages and
behaviour that the examples above don't show. They run in the copy of
[examples/inspect/edge/](../../examples/inspect/edge).

```scrut
$ cd "$TMPDIR/edge"
```

```python
# examples/inspect/edge/main.star
load("starcm", "load_dynamic", "vars", run = "exec")
load("lib.star", "greet")

run(label = "list", cmd = "ls", args = [])
load_dynamic("lib.star", label = "literal path")
load_dynamic(vars["next"], label = "computed path")
greet("ada")
```

```python
# examples/inspect/edge/lib.star
def greet(name):
    print("hello, " + name)
```

```scrut
$ starcm inspect main.star lib.star | jq -c '.[] | {file, loads: [.loads[] | [.module, .names]]}'
{"file":"main.star","loads":[["starcm",{"load_dynamic":"load_dynamic","run":"exec","vars":"vars"}],["lib.star",{"greet":"greet"}]]}
{"file":"lib.star","loads":[]}
```
```scrut
$ starcm inspect main.star | jq -c '.[0].calls[], .[0].values[]'
{"module":"starcm","function":"exec","line":4}
{"module":"starcm","function":"load_dynamic","line":5,"path":"lib.star"}
{"module":"starcm","function":"load_dynamic","line":6,"path":null,"inputs":[{"arg":"0","uses":[{"module":"starcm","function":"vars"}]}]}
{"module":"lib.star","function":"greet","line":7}
{"module":"starcm","function":"vars","line":6}
```

### Nothing runs

```scrut
$ starcm inspect main.star | grep -c hello
0
[1]
```

### Errors

```python
# examples/inspect/edge/broken.star
def (
```

```scrut
$ starcm inspect broken.star 2>&1
inspect: broken.star:1:6: not an identifier
[1]
```

### inspect()

The same report, as a Starlark value.

```python
# examples/inspect/edge/inspect_fn.star
load("starcm", "inspect", "here")

r = inspect(here("main.star"), label = "inspect main.star").value
print([c["function"] for c in r["calls"]])
print([c.get("path") for c in r["calls"] if c["function"] == "load_dynamic"])
print(r["loads"][0]["names"]["run"])
```

```scrut
$ starcm inspect_fn.star
["exec", "load_dynamic", "load_dynamic", "greet"]
["lib.star", None]
exec
```

## See also

- [sign](sign.md): inspect configuration, then sign what passes.
