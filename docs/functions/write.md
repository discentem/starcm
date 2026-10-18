# write

[← All functions](../README.md)

Use `write` to print a value to stdout. Unlike Starlark's built-in `print`,
`write` takes the [common arguments](../common-functionality.md#common-arguments), so a message can
depend on `only_if` / `not_if`, and it returns what it wrote.

## Syntax

```python
load("starcm", "write")

write("hello", label = "greet")             # positional value
write(str = "hello", label = "greet")       # or str=
write("no newline", end = "", label = "greet")
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `str` | any | required | The value to write, positionally or as `str =`. Strings are written as plain text, like `print`; other values as their Starlark representation. |
| `end` | string | `"\n"` | Written after the value. |

`write` also takes the [common arguments](../common-functionality.md#common-arguments): `label`
(required), `only_if`, `not_if`, `timeout`, `working_directory` and
`what_if`. Only `only_if` and `not_if` affect it.

## Returns

A [starcm_result](../common-functionality.md#starcm_result):

| Field | Value |
| --- | --- |
| `success` | `True` |
| `changed` | `False` |
| `value` | The value that was written, unchanged |

## Examples

Each example runs in a scratch copy of
[docs/fixtures/write/](../fixtures/write).

```scrut
$ cp -R "$TESTDIR/../fixtures/write/." "$TMPDIR" && cd "$TMPDIR"
```

### Write values

<!-- fixture: ../fixtures/write/write.star -->
```python
load("starcm", "write")

write("hello from write", label = "greet")
write(str = "keyword form", label = "keyword")
write("no newline -> ", end = "", label = "partial")
write("then newline", label = "rest")
write(["lists", "print", "as", "Starlark"], label = "list")
r = write("written values are returned", label = "returned")
print("value=%s success=%s changed=%s" % (r.value, r.success, r.changed))
```

```scrut
$ starcm write.star
hello from write
keyword form
no newline -> then newline
["lists", "print", "as", "Starlark"]
written values are returned
value=written values are returned success=True changed=False
```

### Write only in some environments

A skipped call logs why it was skipped.

<!-- fixture: ../fixtures/write/conditional.star -->
```python
load("starcm", "write", "vars")

env = vars.get("env", "dev")
write("deploying to production", label = "prod banner", only_if = env == "prod")
write("not production: %s" % env, label = "dev banner", not_if = env == "prod")
```

```scrut
$ starcm conditional.star
INFO: \d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} \[prod banner\]: skipping write\(label="prod banner"\) because only_if was false (regex)
not production: dev
```

```scrut
$ starcm --var env=prod conditional.star
deploying to production
INFO: \d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} \[dev banner\]: skipping write\(label="dev banner"\) because not_if was true (regex)
```

## See also

- [tests/write.md](../../tests/write.md) for argument errors.
