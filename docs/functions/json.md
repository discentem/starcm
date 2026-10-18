# json

[← All functions](../README.md)

`json` is Starlark's standard
[JSON module](https://github.com/google/starlark-go/blob/master/lib/json/json.go),
re-exported by starcm. Use it to read settings files with
[read_file](read_file.md), write them with [file](file.md), or process
the output of `exec` and [inspect](inspect.md). Unlike starcm's other
functions, its functions take no `label` and return plain values, not a
`starcm_result`.

## Syntax

```python
load("starcm", "json")

json.encode(value)                 # Starlark value -> compact JSON string
json.decode(text)                  # JSON string -> Starlark value
json.indent(text, prefix = "", indent = "\t")  # reformat a JSON string
```

## Functions

| Function | Returns | Description |
| --- | --- | --- |
| `json.encode(x)` | string | Encodes dicts (string keys, sorted), lists, tuples, strings, numbers, bools, `None` and structs. |
| `json.decode(x)` | value | Objects become dicts, arrays lists, `null` `None`, numbers ints or floats. Invalid JSON fails the run. |
| `json.indent(x, prefix = "", indent = "\t")` | string | Pretty-prints a JSON string; each line starts with `prefix`, nested levels add `indent`. |

## Examples

Each example runs in a scratch copy of
[docs/fixtures/json/](../fixtures/json).

```scrut
$ cp -R "$TESTDIR/../fixtures/json/." "$TMPDIR" && cd "$TMPDIR"
```

### Encode, decode and indent

<!-- fixture: ../fixtures/json/json.star -->
```python
load("starcm", "json")

doc = json.decode('{"name": "ada", "groups": ["admin", "dev"], "uid": 501, "active": true, "shell": null}')
print(doc["name"], doc["groups"], doc["uid"] + 1, doc["active"], doc["shell"])

print(json.encode({"b": [1, None], "a": True}))
print(json.indent('{"a":1,"b":[1,2]}', indent = "  "))
```

```scrut
$ starcm json.star
ada ["admin", "dev"] 502 True None
{"a":true,"b":[1,null]}
{
  "a": 1,
  "b": [
    1,
    2
  ]
}
```

### Write a JSON settings file

Because `json.encode` sorts keys, the same settings always produce the same
file, and [file](file.md) reports `changed = False` when nothing changed.

<!-- fixture: ../fixtures/json/write_json.star -->
```python
load("starcm", "file", "json")

settings = {"channel": "beta", "auto_update": True, "hosts": ["a", "b"]}
r = file(
    label   = "settings.json",
    path    = "settings.json",
    content = json.indent(json.encode(settings), indent = "  ") + "\n",
)
print("changed=%s" % r.changed)
```

```scrut
$ starcm write_json.star && cat settings.json
changed=True
{
  "auto_update": true,
  "channel": "beta",
  "hosts": [
    "a",
    "b"
  ]
}
```

```scrut
$ starcm write_json.star
changed=False
```

## See also

- [read_file](read_file.md) has an example that decodes a settings file.
- [tests/json.md](../../tests/json.md).
