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
[examples/write/](../../examples/write).

```scrut
$ cp -R "$TESTDIR/../../examples/write/." "$TMPDIR" && cd "$TMPDIR"
```

### Write values

```python
# examples/write/write.star
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

```python
# examples/write/conditional.star
load("starcm", "write", "vars")

env = vars.get("env", "dev")
write("deploying to production", label = "prod banner", only_if = env == "prod")
write("not production: %s" % env, label = "dev banner", not_if = env == "prod")
```

```scrut
$ starcm conditional.star
INFO: [prod banner]: skipping write(label="prod banner") because only_if was false
not production: dev
```

```scrut
$ starcm --var env=prod conditional.star
deploying to production
INFO: [dev banner]: skipping write(label="dev banner") because not_if was true
```

## Errors and edge cases

These are here mostly as tests: CI runs them to check error messages and
behaviour that the examples above don't show. They run in the copy of
[examples/write/edge/](../../examples/write/edge).

```scrut
$ cd "$TMPDIR/edge"
```

### Strings print as plain text, like print()

Other values print their Starlark representation.

```python
# examples/write/edge/plain_text.star
load("starcm", "write")

# Strings print as plain text, like print(): no quotes, escapes applied.
write("hello world", label = "string")
write('she said "hi"', label = "embedded quotes")
write("tab\there", label = "tab")

# Other values print their Starlark representation.
write(42, label = "int")
write(["a", "b"], label = "list")
write({"k": "v"}, label = "dict")
write(None, label = "none")
```

```scrut
$ starcm plain_text.star
hello world
she said "hi"
tab\there (escaped)
42
["a", "b"]
{"k": "v"}
None
```

### Errors

```python
# examples/write/edge/no_value.star
load("starcm", "write")

write(label = "nothing to write")
```

```scrut
$ starcm no_value.star 2>&1
Traceback (most recent call last):
  no_value.star:3:6: in <toplevel>
Error in write: write requires a value, either positionally or as str=
[1]
```

```python
# examples/write/edge/too_many.star
load("starcm", "write")

write("a", "b", label = "two values")
```

```scrut
$ starcm too_many.star 2>&1
Traceback (most recent call last):
  too_many.star:3:6: in <toplevel>
Error in write: write takes at most one positional argument, got 2
[1]
```

```python
# examples/write/edge/both.star
load("starcm", "write")

write("positional", str = "and str", label = "both")
```

```scrut
$ starcm both.star 2>&1
Traceback (most recent call last):
  both.star:3:6: in <toplevel>
Error in write: both: got multiple values for keyword argument "str"
[1]
```

```python
# examples/write/edge/missing_label.star
load("starcm", "write")

write("every starcm function requires a label")
```

```scrut
$ starcm missing_label.star 2>&1
Traceback (most recent call last):
  missing_label.star:3:6: in <toplevel>
Error in write: "label" not found in kwargs
[1]
```

## See also

