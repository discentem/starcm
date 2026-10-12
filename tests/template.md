# template()

```scrut
$ cp -R "$TESTDIR/fixtures/template/." "$TMPDIR" && cd "$TMPDIR"
```

## what_if renders without writing

```scrut
$ starcm what_if.star && test ! -e hello.txt && echo "nothing written"
success=True changed=True
Hello World, you are 42 years old.
nothing written
```

## Renders to destination, idempotently

```scrut
$ starcm render.star
first render: success=True changed=True
same data: success=True changed=False
new data: success=True changed=True
```

```scrut
$ cat hello.txt; echo
Hello World, you are 43 years old.
```

## destination is required unless what_if

```scrut
$ starcm missing_destination.star 2>&1
Traceback (most recent call last):
  missing_destination.star:3:9: in <toplevel>
Error in template: destination is required in template() module if what_if is false
[1]
```
