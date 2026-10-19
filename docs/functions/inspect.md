# inspect

[← All functions](../README.md)

Use `inspect` to find out what a `.star` file loads and calls **without
running it**, so you can enforce rules such as "this file may only use
`download`" before it's signed or deployed. It's available as the
`inspect()` function and as the `starcm inspect` command, which prints the
same report as JSON.

Calls are resolved through `load()` aliases, so
`load("starcm", run = "exec"); run(...)` is reported as a call to `exec`.

## Syntax

```python
load("starcm", "inspect")

inspect("bootstrap.star", label = "check bootstrap").value
inspect(path = "bootstrap.star", label = "check bootstrap").value
```

```bash
starcm inspect FILE...
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `path` | string | required | The `.star` file to inspect, positionally or as `path =`, relative to the directory starcm runs in. |

`inspect` also takes the [common arguments](../common-functionality.md#common-arguments):
`label` (required), `only_if`, `not_if`, `timeout`, `working_directory` and
`what_if`. Only `only_if` and `not_if` affect it.

## Returns

A [starcm_result](../common-functionality.md#starcm_result) with `success =
True`, `changed = False` and `value` set to a report dict. `starcm inspect`
prints a JSON list with one report per file.

| Key | Description |
| --- | --- |
| `file` | The path as given. |
| `loads` | One entry per `load()` statement: `module`, `line`, and `names`, which maps each local name to the name it was loaded as. |
| `calls` | Direct calls to loaded names: `module`, `function` (the original name, after aliases) and `line`. Calls to `load_dynamic` also have `path`: its literal module path, or `None` / `null` when the path is computed at runtime. |
| `values` | Other uses of loaded names, such as `vars["base_url"]` or passing a function around: `module`, `function` and `line`. Calls made through such values can't be known statically. |

A file that can't be read or parsed stops the run (or makes `starcm inspect`
exit 1).

## Examples

Each example runs in a scratch copy of
[docs/fixtures/inspect/](../fixtures/inspect).

```scrut
$ cp -R "$TESTDIR/../fixtures/inspect/." "$TMPDIR" && cd "$TMPDIR"
```

### Report from the command line

<!-- fixture: ../fixtures/inspect/sneaky.star -->
```python
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

<!-- fixture: ../fixtures/inspect/bootstrap.star -->
```python
load("starcm", "download", "load_dynamic", "vars")

download(
    label   = "fetch agent",
    url     = vars["base_url"] + "/agent.pkg",
    save_to = "agent.pkg",
    sha256  = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
)
load_dynamic("platforms/macos.star", label = "macOS settings")
load_dynamic("platforms/%s.star" % vars["platform"], label = "chosen platform")
```

With `jq`, list each call and each `load_dynamic` path:

```scrut
$ starcm inspect bootstrap.star | jq -c '.[].calls[]'
{"module":"starcm","function":"download","line":3}
{"module":"starcm","function":"load_dynamic","line":9,"path":"platforms/macos.star"}
{"module":"starcm","function":"load_dynamic","line":10,"path":null}
```

```scrut
$ starcm inspect bootstrap.star | jq -c '[.[].values[] | [.function, .line]]'
[["vars",5],["vars",10]]
```

### Enforce a policy in Starlark

<!-- fixture: ../fixtures/inspect/policy.star -->
```python
load("starcm", "inspect", "list_files")

ALLOWED = ["download", "load_dynamic"]

for f in list_files(".", pattern = "*.star", label = "find config").value:
    if f == "policy.star":
        continue
    report = inspect(f, label = "inspect " + f).value
    bad = [c["function"] for c in report["calls"] if c["function"] not in ALLOWED]
    print("%s: %s" % (f, "ok" if not bad else "uses %s" % ", ".join(bad)))
```

```scrut
$ starcm policy.star
bootstrap.star: ok
sneaky.star: uses exec
```

## See also

- [sign](sign.md): inspect configuration, then sign what passes.
- [tests/inspect.md](../../tests/inspect.md) for aliases across files and parse errors.
