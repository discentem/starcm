# sign

[← All functions](../README.md)

Use `sign` (or the `starcm sign` command) to sign configuration and payloads
with an ed25519 private key. Each signed file gets a `<file>.sig` next to it.
Machines that run starcm with
[--trusted-keys DIR](../cli.md#--trusted-keys-dir) then refuse any `.star`
file, including ones pulled in by `load()` and
[load_dynamic](load_dynamic.md), that isn't signed by one of those keys, and
can check other files with [verify](verify.md).

Every signature expires. The expiry is part of what's signed, so it can't be
extended by editing the `.sig` file, and an old release stops being accepted
on its own instead of being replayable forever. Runs with
[--max-expiry](../cli.md#--max-expiry-duration) also refuse signatures that
expire too far in the future, so they only trust recently signed files.

Signing belongs on the machine that holds the key, so `sign()` is disabled
when starcm runs with `--trusted-keys`.

## Keys

Generate a key pair with OpenSSL, keep the private key secret, and ship the
public key to the machines that run your configuration:

```bash
openssl genpkey -algorithm ed25519 -out release.pem                 # private key
openssl pkey -in release.pem -pubout -out trusted/release.pub.pem   # public key
```

A key's id is its file name before the first dot (`release` for both files
above). It's recorded in each signature, and `--trusted-keys` looks the public
key up by it.

## Syntax

```python
load("starcm", "sign")

sign("main.star", key = "release.pem", expires = "30d", label = "sign main.star")
sign(path = "main.star", key = "release.pem", expires = "2026-12-31", key_id = "release-2027", label = "sign main.star")
```

```bash
starcm sign --key release.pem --expires 30d [--key-id ID] FILE...
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `path` | string | required | The file to sign, positionally or as `path =`. |
| `key` | string | required | Path to the PEM (PKCS #8) ed25519 private key. `--key` for `starcm sign`. |
| `expires` | string | required | When the signature expires: a duration from now (`"30d"`, `"36h"`), a date (`"2026-11-14"`, midnight UTC) or an RFC 3339 time. `--expires` for `starcm sign`. |
| `key_id` | string | key file name before the first dot | Key id to record in the signature. `--key-id` for `starcm sign`. |

`sign` also takes the [common arguments](../common-functionality.md#common-arguments):
`label` (required), `only_if`, `not_if`, `timeout`, `working_directory` and
`what_if`. Only `only_if` and `not_if` affect it.

## Returns

A [starcm_result](../common-functionality.md#starcm_result):

| Field | Value |
| --- | --- |
| `success` | `True` |
| `changed` | `True` when `<path>.sig` was written or changed. ed25519 signatures are deterministic, so re-signing unchanged content with the same key and a fixed expiry date reports `False`. An expiry given as a duration is counted from now, so it re-signs every time. |
| `message` | `signed <path> with key "<id>" until <expiry> -> <path>.sig` |
| `value` | The key id |

A missing file, an invalid expiry or a key that isn't an ed25519 private key
stops the run.

## Examples

The examples sign with fixed dates so their output doesn't change from run
to run; in practice, a duration such as `30d` with a re-signing schedule is
more typical. They use the test-only key from
[tests/fixtures/signing/](../../tests/fixtures/signing), copied in as
`release.pem` and `trusted/release.pub.pem`. Never trust it outside tests.

```scrut
$ cp -R "$TESTDIR/../fixtures/signing/." "$TMPDIR" && cd "$TMPDIR" && mkdir trusted && cp "$TESTDIR/../../tests/fixtures/signing/private/test.pem" release.pem && cp "$TESTDIR/../../tests/fixtures/signing/trusted/test.pub.pem" trusted/release.pub.pem
```

### Sign configuration with `starcm sign`

```scrut
$ cat main.star lib.star
load("starcm", "write")
load("lib.star", "NAME")

write("hello from signed %s" % NAME, label = "hello")
NAME = "configuration"
```

```scrut
$ starcm sign --expires 2099-01-01 --key release.pem main.star lib.star agent.pkg
signed main.star with key "release" until 2099-01-01T00:00:00Z -> main.star.sig
signed lib.star with key "release" until 2099-01-01T00:00:00Z -> lib.star.sig
signed agent.pkg with key "release" until 2099-01-01T00:00:00Z -> agent.pkg.sig
```

A signature file is one line: the algorithm, the key id, the expiry and the
signature.

```scrut
$ cat main.star.sig
ed25519 key=release expires=2099-01-01T00:00:00Z sig=* (glob)
```

### Run only signed configuration

```scrut
$ starcm --trusted-keys trusted main.star
hello from signed configuration
```

Change a loaded file and the run stops before it executes:

```scrut
$ echo 'NAME = "something else"' > lib.star && starcm --trusted-keys trusted main.star 2>&1
Traceback (most recent call last):
  main.star:2:1: in <toplevel>
Error: cannot load lib.star: signature check failed for lib.star: key "release": signature does not match
[1]
```

### Sign from Starlark

`sign()` can be part of a release script, for example after
[inspect](inspect.md) has checked each file. Here it re-signs every `.star`
file with the same expiry; only the edited `lib.star` changes.

```scrut
$ cat sign_all.star
load("starcm", "list_files", "sign", "vars")

for f in list_files(".", pattern = "*.star", label = "find config").value:
    if f == "sign_all.star":
        continue
    r = sign(f, key = vars["key"], expires = "2099-01-01", label = "sign " + f)
    print("%s: changed=%s key=%s" % (f, r.changed, r.value))
```

```scrut
$ starcm --var key=release.pem sign_all.star
lib.star: changed=True key=release
main.star: changed=False key=release
```

```scrut
$ starcm --trusted-keys trusted main.star
hello from signed something else
```

### Expired signatures are refused

```scrut
$ starcm sign --expires 2020-01-01 --key release.pem main.star && starcm --trusted-keys trusted main.star 2>&1
signed main.star with key "release" until 2020-01-01T00:00:00Z -> main.star.sig
signature check failed for main.star: key "release": signature expired at 2020-01-01T00:00:00Z (now *) (glob)
[1]
```

### Only trust recent signatures with `--max-expiry`

A signature valid until 2099 is refused by a run that only accepts
signatures expiring within 30 days. Re-sign with `--expires 30d` and it runs:

```scrut
$ starcm sign --expires 2099-01-01 --key release.pem main.star >/dev/null && starcm --trusted-keys trusted --max-expiry 30d main.star 2>&1
signature check failed for main.star: key "release": signature expires at 2099-01-01T00:00:00Z, more than 30d from now (*) (glob)
[1]
```

```scrut
$ starcm sign --expires 30d --key release.pem main.star lib.star >/dev/null && starcm --trusted-keys trusted --max-expiry 30d main.star
hello from signed something else
```

### Record a different key id

Name the public key to match, e.g. `trusted/release-2027.pub.pem`.

```scrut
$ starcm sign --expires 30d --key release.pem --key-id release-2027 agent.pkg && cut -d" " -f1,2 agent.pkg.sig
signed agent.pkg with key "release-2027" until * -> agent.pkg.sig (glob)
ed25519 key=release-2027
```

## See also

- [verify](verify.md) to check signatures on downloads and other files.
- [--trusted-keys](../cli.md#--trusted-keys-dir).
- [tests/signing.md](../../tests/signing.md) for untrusted keys, tampering via `load_dynamic`, edited expiries, old-format signatures, bad keys and `sign()` under `--trusted-keys`.
