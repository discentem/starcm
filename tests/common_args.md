# Common arguments

Every function accepts `label`, `only_if`, `not_if`, `timeout`,
`working_directory` and `what_if`. Each call is independent: arguments a call
doesn't pass never keep a value from an earlier call.

```scrut
$ cp -R "$TESTDIR/fixtures/common_args/." "$TMPDIR" && cd "$TMPDIR"
```

## Argument errors name the failing call

The error is prefixed with the failing call's label, not an earlier call's.

```scrut
$ starcm error_prefix.star 2>&1 | grep -v "^  "
first call
Traceback (most recent call last):
Error in write: current: for parameter "end": got int, want string
```

## timeout applies only to its own call

```scrut
$ starcm timeout.star
no timeout: success=True
```

The starcm module is shared by every loaded file, so this holds across files
too: a timeout set in a loaded file doesn't apply to the caller's later calls.

```scrut
$ starcm cross_file.star
main call: success=True
```

## working_directory applies only to its own call

```scrut
$ starcm working_directory.star && echo "sub/out.txt: $(cat sub/out.txt)" && echo "out.txt: $(cat out.txt)"
sub/out.txt: Hello Sub, you are 1 years old.
out.txt: Hello Top, you are 1 years old.
```
