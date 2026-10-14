# read_file()

Returns a file's content as `value`. Like `file()`, the path may start with `~`
and is otherwise relative to the current directory.

```scrut
$ cp -R "$TESTDIR/fixtures/read_file/." "$TMPDIR" && cd "$TMPDIR"
```

```scrut
$ starcm read.star
success=True changed=False
"line one\nline two\n"
2
```

```scrut
$ starcm missing.star 2>&1
Traceback (most recent call last):
  missing.star:2:10: in <toplevel>
Error in read_file: failed to read file "missing\.txt": open .*[/\\]missing\.txt: (no such file or directory|The system cannot find the file specified\.) (regex)
[1]
```
