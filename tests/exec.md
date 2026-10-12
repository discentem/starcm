# exec()

```scrut
$ cp -R "$TESTDIR/fixtures/exec/." "$TMPDIR" && cd "$TMPDIR"
```

## Captures stdout into value

```scrut
$ starcm capture.star
success=True changed=True value="captured\n"
```

## live_output streams stdout

```scrut
$ starcm live_output.star
streamed
```

## expected_exit_code

A non-zero exit is a success when it matches `expected_exit_code`. An
unexpected exit code doesn't stop the run; it's reported in the result.

```scrut
$ starcm exit_codes.star
expected: success=True
unexpected: success=False error=exit status 3
```

## args is required

```scrut
$ starcm missing_args.star 2>&1
Traceback (most recent call last):
  missing_args.star:3:5: in <toplevel>
Error in exec: exec: missing argument for args
[1]
```
