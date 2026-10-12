# write()

Each fixture is copied into scrut's per-document `$TMPDIR` and run from there.

```scrut
$ cp -R "$TESTDIR/fixtures/write/." "$TMPDIR" && cd "$TMPDIR"
```

## Positional argument

`write()` prints its argument's Starlark representation, so strings are quoted.

```scrut
$ starcm positional.star
"hello from write"
success=True changed=False value=None
```

## only_if / not_if

Skipped calls log why; the timestamp varies so those lines are regex-matched.

```scrut
$ starcm conditions.star
"only_if True runs"
INFO: \d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} \[only_if false\]: skipping write\(label="only_if false"\) because only_if was false (regex)
"not_if False runs"
INFO: \d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} \[not_if true\]: skipping write\(label="not_if true"\) because not_if was true (regex)
```

## label is required

```scrut
$ starcm missing_label.star 2>&1
Traceback (most recent call last):
  missing_label.star:3:6: in <toplevel>
Error in write: "label" not found in kwargs
[1]
```
