# examples/

Runs every file under `examples/` so they can't silently fall behind the API.
The directory is copied into scrut's per-document `$TMPDIR`; each example runs
from its own directory (via a subshell) unless noted.

`examples/download/a_file.star` downloads from the internet and is exercised
by README.md instead.

```scrut
$ cp -R "$TESTDIR/../examples" "$TMPDIR" && cd "$TMPDIR/examples"
```

## date

```scrut
$ (cd date && starcm date.star)
local: \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(Z|[+-]\d{2}:\d{2}) (re)
utc date: \d{4}-\d{2}-\d{2} \(\w+day, day \d+ of the year\) (re)
los angeles: \w{3} \w{3} \d{1,2} \d{2}:\d{2} P[SD]T (re)
```

```scrut
$ (cd date && starcm invalid_timezone.star 2>&1)
Traceback (most recent call last):
  invalid_timezone.star:5:5: in <toplevel>
Error in date: invalid timezone "Not/AZone": unknown time zone Not/AZone
[1]
```

## download/load_remote

Downloads from the `python3 -m http.server 8000` the scrut CI job runs from
the repo root. The macro file on its own only defines `load_remote`.

```scrut
$ (cd download/load_remote && starcm load_remote_macro.star)
```

```scrut
$ (cd download/load_remote && starcm load_remote.star && starcm load_remote.star)
hello from e.star
hello from e.star
```

## exec

```scrut
$ (cd exec/exit_codes && starcm expected.star)
we expect to exit 2
starcm_result(changed = True, error = "exit status 2", label = "explicitly exit 2", message = "", success = True, value = "we expect to exit 2\n")
```

```scrut
$ (cd exec/exit_codes && starcm unexpected.star)
we expect to exit 2
starcm_result(changed = True, error = "exit status 2", label = "explicitly exit 2", message = "", success = False, value = "we expect to exit 2\n")
```

```scrut
$ (cd exec && starcm expect_timeout_shell.star)
starcm_result(changed = False, error = "context deadline exceeded", label = "sleep for 5s, timeout after 3", message = "", success = False, value = None)
```

## files

`~` expands to `$HOME`, pointed at `$TMPDIR` so the example doesn't write to
the real home directory.

```scrut
$ (cd files && HOME="$TMPDIR" starcm create.star)
starcm_result(changed = True, error = "<nil>", label = "Create example.txt file", message = "created file \"*example.txt\"", success = True, value = None) (glob)
```

```scrut
$ (cd files && HOME="$TMPDIR" starcm create.star)
starcm_result(changed = False, error = "<nil>", label = "Create example.txt file", message = "file \"*example.txt\" already exists with correct content", success = True, value = None) (glob)
```

```scrut
$ cat "$TMPDIR/example.txt"
This is an example file created by starcm.
```

## if_statements

```scrut
$ (cd if_statements && starcm if_statements.star)
party!
```

## load_from_loop

```scrut
$ (cd load_from_loop && starcm loop.star)
hello from e.star
hello from d.star
```

## nested_files

```scrut
$ (cd nested_files && starcm a.star)
hello from a.star
INFO: * [load b.star]: skipping load(label="load b.star") because only_if was false (glob)
```

```scrut
$ (cd nested_files && starcm e.star)
hello from e.star
```

```scrut
$ (cd nested_files/subfolder/subfolder2/subfolder3 && starcm d.star)
hello from d.star
```

`b.star` and `c.star` load `//examples/nested_files/e.star`, which resolves
against the workspace (the current directory), so they run from the root.

```scrut
$ (cd .. && starcm examples/nested_files/subfolder/b.star)
hello from b.star
hello from c.star
hello from d.star
hello from e.star
```

```scrut
$ (cd .. && starcm examples/nested_files/subfolder/subfolder2/c.star)
hello from c.star
hello from d.star
hello from e.star
```

## only_if

```scrut
$ (cd only_if && starcm only_if.star)
we expect to exit 2
INFO: * [print_not_success_#2]: skipping write(label="print_not_success_#2") because only_if was false (glob)
```

## templates

```scrut
$ (cd templates/simple && starcm template.star)
Hello World, you are 42 years old.
```

```scrut
$ (cd templates/nested && starcm nested.star)
[server]
host = localhost
port = 8080

[user.ada]
groups = admin, dev
shell = /bin/zsh

[user.bob]
groups = dev

[build]
date = \d{4}-\d{2}-\d{2} (re)
```

## write

```scrut
$ (cd write && starcm write.star)
hello world from write!
```

## Every example is covered

Fails when an example is added without a test above.

```scrut
$ (cd "$TESTDIR/../examples" && find . -name '*.star' | sort)
./date/date.star
./date/invalid_timezone.star
./download/a_file.star
./download/load_remote/load_remote.star
./download/load_remote/load_remote_macro.star
./exec/exit_codes/expected.star
./exec/exit_codes/unexpected.star
./exec/expect_timeout_shell.star
./files/create.star
./if_statements/if_statements.star
./load_from_loop/loop.star
./nested_files/a.star
./nested_files/e.star
./nested_files/subfolder/b.star
./nested_files/subfolder/subfolder2/c.star
./nested_files/subfolder/subfolder2/subfolder3/d.star
./only_if/only_if.star
./templates/nested/nested.star
./templates/simple/template.star
./write/write.star
```
