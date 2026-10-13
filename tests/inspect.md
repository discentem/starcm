# starcm inspect

Prints what files load and call, as JSON, without running them. Calls are
resolved through `load()` aliases; `path` is `load_dynamic`'s literal path, or
`null` when it's computed at runtime; `values` are loaded names used other
than by a direct call.

```scrut
$ cp -R "$TESTDIR/fixtures/inspect/." "$TMPDIR" && cd "$TMPDIR"
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
{"module":"starcm","function":"load_dynamic","line":6,"path":null}
{"module":"lib.star","function":"greet","line":7}
{"module":"starcm","function":"vars","line":6}
```

## Nothing runs

```scrut
$ starcm inspect main.star | grep -c hello
0
[1]
```

## Errors

```scrut
$ starcm inspect broken.star 2>&1
inspect: broken.star:1:6: not an identifier
[1]
```

## inspect()

The same report, as a Starlark value.

```scrut
$ starcm inspect_fn.star
["exec", "load_dynamic", "load_dynamic", "greet"]
["lib.star", None]
exec
```
