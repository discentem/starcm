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

The file ends the way the template does, including its trailing newline.

```scrut
$ starcm render.star
first render: success=True changed=True
same data: success=True changed=False
new data: success=True changed=True
```

```scrut
$ cat hello.txt
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

## Nested data

`data` can hold nested dicts, lists, tuples, `None` and structs, including
results from other starcm functions. Dict loops (`for k, v in d`) run in
alphabetical key order; use a list when order matters.

```scrut
$ starcm nested.star
hello
dots: ada lives in london
brackets: ada rex
list: cat=tom dog=rex 
dict: ALPHA=2 ZED=1 
tuple: 3,4
none: [] unset
struct: hello success=True
```

## Dict keys must be strings

```scrut
$ starcm non_string_key.star 2>&1
Traceback (most recent call last):
  non_string_key.star:3:9: in <toplevel>
Error in template: template() data: "age": dict keys must be strings, got int key 1
[1]
```
