# --var

`--var name=value` (repeatable) passes values into a run as the `vars` dict.
Values are kept whole, commas included, and keys are sorted.

```scrut
$ cp -R "$TESTDIR/fixtures/vars/." "$TMPDIR" && cd "$TMPDIR"
```

```scrut
$ starcm --var base=http://localhost:8000/stable --var list=a,b --var empty= vars.star
{"base": "http://localhost:8000/stable", "empty": "", "list": "a,b"}
base=http://localhost:8000/stable
```

```scrut
$ starcm vars.star
{}
base=unset
```

```scrut
$ starcm --var novalue vars.star 2>&1
--var "novalue": want name=value
[1]
```
