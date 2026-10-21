# Command line

[← Documentation index](README.md)

```bash
starcm [--var name=value]... [--trusted-keys DIR [--max-expiry DURATION] [--release ID]] [--store-dir DIR] [--store-only] [-v N] [--timestamps] FILE
starcm sign --key KEY --expires WHEN [--key-id ID] [--release ID] FILE...
starcm inspect FILE...
```

`starcm FILE` runs one `.star` file, and every file it loads, in the current
directory. It exits 0 when the file runs to the end and 1 when the run stops
with an error (a failing function, `fail()`, a refused signature, a bad flag).
Results that only report failure, such as an `exec` with an unexpected exit
code, don't change the exit code unless the configuration calls `fail()`.

## Options

| Option | Default | Description |
| --- | --- | --- |
| [--var name=value](#--var-namevalue) | none | Pass a value to the run as `vars["name"]`. Repeatable. |
| [--trusted-keys DIR](#--trusted-keys-dir) | none | Require every `.star` file to be signed by a key in `DIR`, and enable `verify()`. |
| [--max-expiry DURATION](#--max-expiry-duration) | none | With `--trusted-keys`, also refuse signatures that expire more than `DURATION` (e.g. `30d`, `36h`) from now. |
| [--release ID](#--release-id) | none | With `--trusted-keys`, refuse signatures that weren't made for release `ID`. |
| [--store-dir DIR](#--store-dir-dir-and---store-only) | see below | Where `download()` keeps files when `save_to` is omitted. |
| [--store-only](#--store-dir-dir-and---store-only) | `false` | Refuse `download(save_to = ...)`, so every download goes into the store. |
| `-v N` | `1` | Log verbosity. Higher values log more of starcm's internals; `3` or more traces every function call. |
| `--timestamps` | `false` | Start every log line with the date and time, in UTC. |
| `--help`, `-h` | | Show help. Running `starcm` with no file does the same. |

Options go before the file name.

## Commands

| Command | Description |
| --- | --- |
| [starcm sign](functions/sign.md) | Sign files, writing `<file>.sig` next to each. |
| [starcm inspect](functions/inspect.md) | Print what `.star` files load and call, as JSON, without running them. |

## Examples

```scrut
$ cp -R "$TESTDIR/../examples/cli/." "$TMPDIR" && cd "$TMPDIR"
```

```python
# examples/cli/hello.star
load("starcm", "vars", "write")

write("hello, %s" % vars.get("name", "world"), label = "greet")
```

### `--var name=value`

Values are strings, kept whole (commas included). See
[vars](functions/vars.md).

```scrut
$ starcm hello.star
hello, world
```

```scrut
$ starcm --var name=fleet hello.star
hello, fleet
```

### `--trusted-keys DIR`

`DIR` holds PEM public keys named `<key id>.<anything>.pem`, e.g.
`release.pub.pem` for key id `release`. With it, starcm checks each `.star`
file's `<file>.sig` before running it, including files loaded with `load()`
and `load_dynamic()`, and stops at the first one that isn't signed by a
trusted key. [verify()](functions/verify.md) checks other files against the
same keys, and [sign()](functions/sign.md) is disabled. See
[sign](functions/sign.md) for making keys and signatures.

```scrut
$ mkdir trusted && cp "$TESTDIR/keys/trusted/test.pub.pem" trusted/release.pub.pem && starcm --trusted-keys trusted hello.star 2>&1
signature check failed for hello.star: no signature file hello.star.sig
[1]
```

```scrut
$ starcm sign --expires 30d --key "$TESTDIR/keys/private/test.pem" --key-id release hello.star && starcm --trusted-keys trusted hello.star
signed hello.star with key "release" until * -> hello.star.sig (glob)
hello, world
```

### `--max-expiry DURATION`

Every signature expires (see [sign](functions/sign.md)). `--max-expiry` also
refuses signatures that expire more than `DURATION` from now, so a run only
trusts files that were signed recently; re-sign on a schedule even when
nothing changed. `DURATION` is a number of days (`30d`) or a Go duration
(`36h`). It needs `--trusted-keys`.

```scrut
$ starcm --trusted-keys trusted --max-expiry 60d hello.star
hello, world
```

```scrut
$ starcm --trusted-keys trusted --max-expiry 7d hello.star 2>&1
signature check failed for hello.star: key "release": signature expires at *, more than 7d from now (*) (glob)
[1]
```

```scrut
$ starcm --max-expiry 30d hello.star 2>&1
--max-expiry needs --trusted-keys
[1]
```

### `--release ID`

`starcm sign --release ID` records a release in each signature, and the
release is covered by the signature. With `--release ID`, starcm refuses any
signature made for a different release, or for none. A server then can't
hand out files from an older release whose signatures haven't expired yet, or
mix files from two releases. It needs `--trusted-keys`.

Give the release from somewhere the server can't change, such as the
launchd job or scheduled task that runs starcm.

```scrut
$ starcm --trusted-keys trusted --release 2026.10 hello.star 2>&1
signature check failed for hello.star: key "release": signature has no release, want release "2026.10"
[1]
```

```scrut
$ starcm sign --expires 30d --key "$TESTDIR/keys/private/test.pem" --key-id release --release 2026.10 hello.star && starcm --trusted-keys trusted --release 2026.10 hello.star
signed hello.star with key "release" for release "2026.10" until * -> hello.star.sig (glob)
hello, world
```

```scrut
$ starcm --trusted-keys trusted --release 2026.11 hello.star 2>&1
signature check failed for hello.star: key "release": signature is for release "2026.10", want release "2026.11"
[1]
```

```scrut
$ starcm --release 2026.10 hello.star 2>&1
--release needs --trusted-keys
[1]
```

### `--store-dir DIR` and `--store-only`

Without `save_to`, [download()](functions/download.md#the-local-store) saves
files into a local store at `DIR/sha256/<sha256>/<name>`, and
[exec()](functions/exec.md#files-from-the-local-store) re-checks stored files
before running them. Without `--store-dir`, the store is:

| Running as | macOS | Linux | Windows |
| --- | --- | --- | --- |
| root / elevated | `/Library/Application Support/starcm/store` | `/var/lib/starcm/store` | `%ProgramData%\starcm\store` |
| anyone else | `~/Library/Caches/starcm/store` | `$XDG_CACHE_HOME/starcm/store` (`~/.cache/...`) | `%LocalAppData%\starcm\store` |

starcm refuses a store, or any directory above it, that anyone other than the
current user, root or administrators could change. `--store-only` makes
`download(save_to = ...)` an error, so every download goes through the store.
[download](functions/download.md) has examples of both.

The store is only created when a run uses it:

```scrut
$ starcm --store-dir "$TMPDIR/unused-store" hello.star && test ! -e "$TMPDIR/unused-store" && echo "no store was created"
hello, world
no store was created
```

### Verbosity and logs

Skipped calls are always logged:

```python
# examples/cli/skip.star
load("starcm", "write")

write("skipped", label = "skip me", only_if = False)
```

```scrut
$ starcm skip.star
INFO: [skip me]: skipping write(label="skip me") because only_if was false
```

`-v 3` adds a line before and after every function call:

```scrut
$ starcm -v 3 hello.star
INFO: [LoadFromFile]: loading file "hello.star"
INFO: [base.go]: calling m.Action.Run(ctx, workingDirectory=".", label="greet", args, kwargs)
hello, world
INFO: [base.go]: finished m.Action.Run for label="greet"
```

`--timestamps` starts each log line with the date and time, in UTC:

```scrut
$ starcm --timestamps skip.star
INFO: \d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} \[skip me\]: skipping write\(label="skip me"\) because only_if was false (regex)
```

### Errors

```scrut
$ starcm missing.star 2>&1
open missing\.star: (no such file or directory|The system cannot find the file specified\.) (regex)
[1]
```

## See also

- [vars](functions/vars.md), [sign](functions/sign.md) and [verify](functions/verify.md).

