# Command line

[← Documentation index](README.md)

```bash
starcm [--var name=value]... [--trusted-keys DIR [--max-expiry DURATION]] [--store-dir DIR] [--store-only] [-v N] [--timestamps] FILE
starcm sign --key KEY --expires WHEN [--key-id ID] FILE...
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
| [--store-dir DIR](#--store-dir-dir-and---store-only) | see below | Where `download()` keeps files when `save_to` is omitted. |
| [--store-only](#--store-dir-dir-and---store-only) | `false` | Refuse `download(save_to = ...)`, so every download goes into the store. |
| `-v N` | `1` | Log verbosity. Higher values log more of starcm's internals; `3` or more traces every function call. |
| `--timestamps` | `false` | Log times in UTC. Log lines always have a timestamp; without this flag it's in local time. |
| `--help`, `-h` | | Show help. Running `starcm` with no file does the same. |

Options go before the file name.

## Commands

| Command | Description |
| --- | --- |
| [starcm sign](functions/sign.md) | Sign files, writing `<file>.sig` next to each. |
| [starcm inspect](functions/inspect.md) | Print what `.star` files load and call, as JSON, without running them. |

## Examples

```scrut
$ cp -R "$TESTDIR/fixtures/cli/." "$TMPDIR" && cd "$TMPDIR"
```

<!-- fixture: fixtures/cli/hello.star -->
```python
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
$ mkdir trusted && cp "$TESTDIR/../tests/fixtures/signing/trusted/test.pub.pem" trusted/release.pub.pem && starcm --trusted-keys trusted hello.star 2>&1
signature check failed for hello.star: no signature file hello.star.sig
[1]
```

```scrut
$ starcm sign --expires 30d --key "$TESTDIR/../tests/fixtures/signing/private/test.pem" --key-id release hello.star && starcm --trusted-keys trusted hello.star
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

<!-- fixture: fixtures/cli/skip.star -->
```python
load("starcm", "write")

write("skipped", label = "skip me", only_if = False)
```

```scrut
$ starcm skip.star
INFO: \d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} \[skip me\]: skipping write\(label="skip me"\) because only_if was false (regex)
```

`-v 3` adds a line before and after every function call:

```scrut
$ starcm -v 3 hello.star
INFO: \d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} \[LoadFromFile\]: loading file "hello.star" (regex)
INFO: \d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} \[base.go\]: calling m.Action.Run\(ctx, workingDirectory=".", label="greet", args, kwargs\) (regex)
hello, world
INFO: \d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} \[base.go\]: finished m.Action.Run for label="greet" (regex)
```

### Errors

```scrut
$ starcm missing.star 2>&1
open missing\.star: (no such file or directory|The system cannot find the file specified\.) (regex)
[1]
```

## See also

- [tests/vars.md](../tests/vars.md) and [tests/signing.md](../tests/signing.md).
