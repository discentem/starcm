# list_files()

Lists the files under a directory, recursively, as sorted paths relative to it
(forward slashes on every platform). `pattern` matches file names.

```scrut
$ cp -R "$TESTDIR/fixtures/list_files/." "$TMPDIR" && cd "$TMPDIR"
```

```scrut
$ starcm list.star
["lib/winpe.star", "main.star", "notes.txt"]
["lib/winpe.star", "main.star"]
["winpe.star"]
```

```scrut
$ starcm bad_pattern.star 2>&1
Traceback (most recent call last):
  bad_pattern.star:2:11: in <toplevel>
Error in list_files: invalid pattern "[": syntax error in pattern
[1]
```
