# file()

```scrut
$ cp -R "$TESTDIR/fixtures/file/." "$TMPDIR" && cd "$TMPDIR"
```

## create is idempotent

```scrut
$ starcm create.star
create: success=True changed=True
same content: success=True changed=False
new content: success=True changed=True
```

```scrut
$ cat out.txt
v2
```

## create_dirs

```scrut
$ starcm create_dirs.star && cat a/b/c.txt
success=True changed=True
deep
```

```scrut
$ starcm missing_dir.star 2>&1
Traceback (most recent call last):
  missing_dir.star:3:5: in <toplevel>
Error in file: directory ".*/missing" does not exist and create_dirs is false (regex)
[1]
```

## delete

```scrut
$ starcm delete.star && test ! -e gone.txt && echo deleted
create: success=True changed=True
delete: success=True changed=True
delete again: success=True changed=False
deleted
```
