# json

Starlark's standard `json` module: `encode`, `decode` and `indent`.

```scrut
$ cp -R "$TESTDIR/fixtures/json/." "$TMPDIR" && cd "$TMPDIR"
```

```scrut
$ starcm json.star
ada ["a", "b"] 4 True None
{"a":true,"b":[1,null]}
{
	"a": 1
}
```
