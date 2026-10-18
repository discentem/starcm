# write()

Each fixture is copied into scrut's per-document `$TMPDIR` and run from there.

```scrut
$ cp -R "$TESTDIR/fixtures/write/." "$TMPDIR" && cd "$TMPDIR"
```

## Positional value

The written value is also returned as `value`.

```scrut
$ starcm positional.star
hello from write
success=True changed=False value=hello from write
```

## str= keyword

```scrut
$ starcm str_kwarg.star
hello from str
success=True changed=False value=hello from str
```

## end= works with either form

```scrut
$ starcm end.star
no newline -> then newline
str + end!
```

## Strings print as plain text, like print()

Other values print their Starlark representation.

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

## only_if / not_if

Skipped calls log why.

```scrut
$ starcm conditions.star
only_if True runs
INFO: [only_if false]: skipping write(label="only_if false") because only_if was false
not_if False runs
INFO: [not_if true]: skipping write(label="not_if true") because not_if was true
```

## Errors

```scrut
$ starcm no_value.star 2>&1
Traceback (most recent call last):
  no_value.star:3:6: in <toplevel>
Error in write: write requires a value, either positionally or as str=
[1]
```

```scrut
$ starcm too_many.star 2>&1
Traceback (most recent call last):
  too_many.star:3:6: in <toplevel>
Error in write: write takes at most one positional argument, got 2
[1]
```

```scrut
$ starcm both.star 2>&1
Traceback (most recent call last):
  both.star:3:6: in <toplevel>
Error in write: both: got multiple values for keyword argument "str"
[1]
```

```scrut
$ starcm missing_label.star 2>&1
Traceback (most recent call last):
  missing_label.star:3:6: in <toplevel>
Error in write: "label" not found in kwargs
[1]
```
