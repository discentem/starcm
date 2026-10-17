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

load_dynamic("roles/web.star", label = "load web role")
load_dynamic(module = "roles/web.star", label = "load web role")
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `module` | string | required | Path of the `.star` file, positionally or as `module =`. Paths starting with `//` are relative to the directory starcm runs in; other relative paths are relative to the calling file. See [Loading files](../loading.md). |

`load_dynamic` also takes the [common arguments](../common-functionality.md#common-arguments):
`label` (required), `only_if`, `not_if`, `timeout`, `working_directory` and
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
[docs/fixtures/load_dynamic/](../fixtures/load_dynamic), which has two
roles:

```scrut
$ cp -R "$TESTDIR/../fixtures/load_dynamic/." "$TMPDIR" && cd "$TMPDIR"
```

```scrut
$ cat roles/web.star
load("starcm", "write")

PORT = 443

def configure():
    write("configuring web server on port %d" % PORT, label = "web")
```

### Pick a module with `--var`

```scrut
$ cat by_role.star
load("starcm", "load_dynamic", "vars")

role = vars.get("role", "web")
r = load_dynamic("roles/%s.star" % role, label = "load role %s" % role)
print("success=%s changed=%s" % (r.success, r.changed))
print(sorted(r.value.keys()))
r.value["configure"]()
```

```scrut
$ starcm by_role.star
success=True changed=False
["PORT", "configure"]
configuring web server on port 443
```

```scrut
$ starcm --var role=db by_role.star
success=True changed=False
["PORT", "configure"]
configuring database on port 5432
```

### Load modules in a loop

```scrut
$ cat all_roles.star
load("starcm", "load_dynamic")

for role in ["db", "web"]:
    mod = load_dynamic(module = "roles/%s.star" % role, label = "load " + role).value
    print("%s listens on %d" % (role, mod["PORT"]))
```

```scrut
$ starcm all_roles.star
db listens on 5432
web listens on 443
```

### Load a module only when it's needed

A skipped call returns `value = None`.

```scrut
$ cat optional.star
load("starcm", "load_dynamic", "vars")

r = load_dynamic(
    "roles/db.star",
    label   = "database role",
    only_if = vars.get("with_db") == "yes",
)
if r.value:
    r.value["configure"]()
else:
    print("database role skipped")
```

```scrut
$ starcm optional.star
INFO: \d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} \[database role\]: skipping load\(label="database role"\) because only_if was false (regex)
database role skipped
```

```scrut
$ starcm --var with_db=yes optional.star
configuring database on port 5432
```

## See also

- [Loading files](../loading.md) for `load()` and how paths resolve.
- [inspect](inspect.md) reports each `load_dynamic` call's path, or `null` when it's computed.
- [tests/load_dynamic.md](../../tests/load_dynamic.md) for missing modules.
