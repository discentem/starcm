# struct

[← All functions](../README.md)

`struct` builds an immutable record with named fields, the same kind of value
starcm's functions return. It comes from the `stdlib` module, not `starcm`.
Use it to group related settings and pass them to functions or
[template](template.md) data.

## Syntax

```python
load("stdlib", "struct")

app = struct(name = "agent", version = "1.2.0")
app.name      # "agent"
```

## Arguments

Any keyword arguments; each becomes a field. Fields are read with `.name`,
and reading a missing field fails the run. `print` shows fields sorted by
name.

## Examples

```scrut
$ cp -R "$TESTDIR/../../examples/struct/." "$TMPDIR" && cd "$TMPDIR"
```

```python
# examples/struct/struct.star
load("starcm", "write")
load("stdlib", "struct")

app = struct(name = "agent", version = "1.2.0", ports = [443, 8443])
print(app.name, app.version, app.ports)
print(app)
write("%s %s" % (app.name, app.version), label = "use fields")
```

```scrut
$ starcm struct.star
agent 1.2.0 [443, 8443]
struct(name = "agent", ports = [443, 8443], version = "1.2.0")
agent 1.2.0
```

## See also

- [Loading files](../loading.md) for the `starcm` and `stdlib` modules.
