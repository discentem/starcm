# load_dynamic()

```scrut
$ cp -R "$TESTDIR/fixtures/load_dynamic/." "$TMPDIR" && cd "$TMPDIR"
```

## Returns the module's globals as a dict

```scrut
$ starcm positional.star
success=True changed=False
["GREETING", "greet"]
hello, starcm
```

## module= keyword

```scrut
$ starcm module_kwarg.star
hello
```

## Computed paths

Unlike `load()`, the path can be built at runtime.

```scrut
$ starcm loop.star
hello
```

## Missing module fails the run

```scrut
$ starcm missing.star 2>&1
Traceback (most recent call last):
  missing.star:3:13: in <toplevel>
Error in load_dynamic: loading module "nope\.star": open nope\.star: (no such file or directory|The system cannot find the file specified\.) (regex)
[1]
```
