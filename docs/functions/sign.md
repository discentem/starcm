# sign

[← All functions](../README.md)

Use `sign` (or the `starcm sign` command) to sign configuration and payloads
with an ed25519 private key. Each signed file gets a `<file>.sig` next to it.
Machines that run starcm with
[--trusted-keys DIR](../cli.md#--trusted-keys-dir) then refuse any `.star`
file, including ones pulled in by `load()` and
[load_dynamic](load_dynamic.md), that isn't signed by one of those keys, and
can audit other files with [verify_all](verify_all.md).

Every signature expires. The expiry is part of what's signed, so it can't be
extended by editing the `.sig` file, and an old release stops being accepted
on its own instead of being replayable forever. Runs with
[--max-expiry](../cli.md#--max-expiry-duration) also refuse signatures that
expire too far in the future, so they only trust recently signed files.

A signature can also name the release it was signed for. Runs with
[--release](../cli.md#--release-id) refuse files signed for any other
release, so a server can't replay an older release, or mix in a file from
one, while its signatures are still unexpired.

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
load("starcm", "here", "sign")

sign(here("main.star"), key = "~/keys/release.pem", expires = "30d", label = "sign main.star")
sign(path = here("main.star"), key = "~/keys/release.pem", expires = "2026-12-31", key_id = "release-2027", release = "2026.10", label = "sign main.star")
```

```bash
starcm sign --key release.pem --expires 30d [--key-id ID] [--release ID] FILE...
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `path` | string | required | The file to sign, positionally or as `path =`. An absolute path; a leading `~` expands to the home directory. |
| `key` | string | required | Absolute path to the PEM (PKCS #8) ed25519 private key; `~` expands. `--key` for `starcm sign`. |
| `expires` | string | required | When the signature expires: a duration from now (`"30d"`, `"36h"`), a date (`"2026-11-14"`, midnight UTC) or an RFC 3339 time. `--expires` for `starcm sign`. |
| `key_id` | string | key file name before the first dot | Key id to record in the signature. `--key-id` for `starcm sign`. |
| `release` | string | none | The release to sign for, with no spaces, e.g. `"2026.10"`. Runs with [--release](../cli.md#--release-id) only accept files signed for their release. `--release` for `starcm sign`. |

`sign` also takes the [common arguments](../common-functionality.md#common-arguments):
`label` (required), `only_if`, `not_if`, `timeout` and
`what_if`. Only `only_if` and `not_if` affect it.

## Returns

A [starcm_result](../common-functionality.md#starcm_result):

| Field | Value |
| --- | --- |
| `success` | `True` |
| `changed` | `True` when `<path>.sig` was written or changed. ed25519 signatures are deterministic, so re-signing unchanged content with the same key and a fixed expiry date reports `False`. An expiry given as a duration is counted from now, so it re-signs every time. |
| `message` | `signed <path> with key "<id>" until <expiry> -> <path>.sig`, with ` for release "<release>"` after the key when there is one |
| `value` | The key id |

A missing file, an invalid expiry or a key that isn't an ed25519 private key
stops the run.

## Examples

The examples sign with fixed dates so their output doesn't change from run
to run; in practice, a duration such as `30d` with a re-signing schedule is
more typical. They use the test-only key from
[docs/keys/](../keys), copied in as
`release.pem` and `trusted/release.pub.pem`. Never trust it outside tests.

```scrut
$ cp -R "$TESTDIR/../../examples/signing/." "$TMPDIR" && cd "$TMPDIR" && mkdir trusted && cp "$TESTDIR/../keys/private/test.pem" release.pem && cp "$TESTDIR/../keys/trusted/test.pub.pem" trusted/release.pub.pem
```

### Sign configuration with `starcm sign`

```python
# examples/signing/main.star
load("starcm", "write")
load("lib.star", "NAME")

write("hello from signed %s" % NAME, label = "hello")
```

```python
# examples/signing/lib.star
NAME = "configuration"
```

```scrut
$ starcm sign --expires 2099-01-01 --key release.pem main.star lib.star agent.pkg
signed main.star with key "release" until 2099-01-01T00:00:00Z -> main.star.sig
signed lib.star with key "release" until 2099-01-01T00:00:00Z -> lib.star.sig
signed agent.pkg with key "release" until 2099-01-01T00:00:00Z -> agent.pkg.sig
```

A signature file is one line: the algorithm, the key id, the expiry, the
release when there is one, the signed file's sha256 and the signature. All of
them are covered by the signature, so none can be changed without breaking
it.

```scrut
$ cat main.star.sig
ed25519 key=release expires=2099-01-01T00:00:00Z sha256=b7b0ba01565b10e6e5f6cd227e2bb72b188437d0d57690c9f0ced3725d2aaaa5 sig=* (glob)
```

```scrut
$ (command -v sha256sum >/dev/null && sha256sum main.star || shasum -a 256 main.star) | cut -d" " -f1
b7b0ba01565b10e6e5f6cd227e2bb72b188437d0d57690c9f0ced3725d2aaaa5
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

```python
# examples/signing/sign_all.star
load("starcm", "here", "path", "sign", "vars")

for f in path.list(here(), pattern = "*.star"):
    # Only this folder's configuration files, not this script or subfolders.
    if f == "sign_all.star" or "/" in f:
        continue
    r = sign(here(f), key = vars["key"], expires = "2099-01-01", label = "sign " + f)
    print("%s: changed=%s key=%s" % (f, r.changed, r.value))
```

```scrut
$ starcm --var key="$(pwd -W 2>/dev/null || pwd)/release.pem" sign_all.star
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

### Sign for a release with `--release`

Sign a release's files together, and give each run the release it should
accept:

```scrut
$ starcm sign --expires 30d --key release.pem --release 2026.10 main.star lib.star && starcm --trusted-keys trusted --release 2026.10 main.star
signed main.star with key "release" for release "2026.10" until * -> main.star.sig (glob)
signed lib.star with key "release" for release "2026.10" until * -> lib.star.sig (glob)
hello from signed something else
```

```scrut
$ cat main.star.sig
ed25519 key=release expires=* release=2026.10 sha256=* sig=* (glob)
```

A `lib.star` from last month's release is still validly signed and hasn't
expired, but the run refuses it:

```scrut
$ echo 'NAME = "last month"' > lib.star && starcm sign --expires 30d --key release.pem --release 2026.09 lib.star >/dev/null && starcm --trusted-keys trusted --release 2026.10 main.star 2>&1
Traceback (most recent call last):
  main.star:2:1: in <toplevel>
Error: cannot load lib.star: signature check failed for lib.star: key "release": signature is for release "2026.09", want release "2026.10"
[1]
```

### Record a different key id

Name the public key to match, e.g. `trusted/release-2027.pub.pem`.

```scrut
$ starcm sign --expires 30d --key release.pem --key-id release-2027 agent.pkg && cut -d" " -f1,2 agent.pkg.sig
signed agent.pkg with key "release-2027" until * -> agent.pkg.sig (glob)
ed25519 key=release-2027
```

## Errors and edge cases

These are here mostly as tests: CI runs them to check error messages and
behaviour that the examples above don't show. They run in the copy of
[examples/signing/edge/](../../examples/signing/edge).

```scrut
$ cd "$TMPDIR/edge" && cp -R "$TESTDIR/../keys/." .
```

### Signing

The key id defaults to the key file's name before the first dot.

```python
# examples/signing/edge/main.star
load("starcm", "load_dynamic", "vars", "write")
load("lib.star", "greet")

write(greet(vars.get("name", "nobody")), label = "greet")
load_dynamic("dyn.star", label = "load dyn.star")
```

```python
# examples/signing/edge/lib.star
def greet(name):
    return "hello, " + name
```

```python
# examples/signing/edge/dyn.star
print("dyn.star ran")
```

```python
# examples/signing/edge/verify.star
load("starcm", "here", "verify_all")

v = verify_all([here("payload.txt")])[0]
print("signed: key_id=%s release=%r" % (v.key_id, v.release))
```

```python
# examples/signing/edge/verify_bad.star
load("starcm", "here", "verify_all")

verify_all([here("payload.txt"), here("tampered.txt"), here("unsigned.txt")])
```

```scrut
$ starcm sign --expires 30d --key private/test.pem main.star lib.star dyn.star verify.star verify_bad.star payload.txt
signed main.star with key "test" until * -> main.star.sig (glob)
signed lib.star with key "test" until * -> lib.star.sig (glob)
signed dyn.star with key "test" until * -> dyn.star.sig (glob)
signed verify.star with key "test" until * -> verify.star.sig (glob)
signed verify_bad.star with key "test" until * -> verify_bad.star.sig (glob)
signed payload.txt with key "test" until * -> payload.txt.sig (glob)
```

```scrut
$ cat main.star.sig
ed25519 key=test expires=* sha256=* sig=* (glob)
```

### Signed files run

`main.star` loads `lib.star` with `load()` and `dyn.star` with `load_dynamic()`.

```scrut
$ starcm --trusted-keys trusted main.star
hello, nobody
dyn.star ran
```

### Without --trusted-keys nothing is checked

```python
# examples/signing/edge/unsigned.star
print("unsigned.star ran")
```

```scrut
$ starcm unsigned.star
unsigned.star ran
```

### Unsigned files are refused

```scrut
$ starcm --trusted-keys trusted unsigned.star 2>&1
signature check failed for unsigned.star: no signature file unsigned.star.sig
[1]
```

### Edited files are refused, whichever way they're loaded

```scrut
$ cp lib.star lib.orig && echo "# edited" >> lib.star && starcm --trusted-keys trusted main.star 2>&1
Traceback (most recent call last):
  main.star:2:1: in <toplevel>
Error: cannot load lib.star: signature check failed for lib.star: key "test": signature does not match
[1]
```

`main.star` runs up to the `load_dynamic()` call before stopping.

```scrut
$ mv lib.orig lib.star && cp dyn.star dyn.orig && echo "# edited" >> dyn.star && starcm --trusted-keys trusted main.star 2>&1
hello, nobody
Traceback (most recent call last):
  main.star:5:13: in <toplevel>
Error in load_dynamic: signature check failed for *dyn.star: key "test": signature does not match (glob)
[1]
```

```scrut
$ mv dyn.orig dyn.star && starcm --trusted-keys trusted main.star
hello, nobody
dyn.star ran
```

### Signatures from keys that aren't trusted are refused

```scrut
$ starcm sign --expires 30d --key private/untrusted.pem unsigned.star && starcm --trusted-keys trusted unsigned.star 2>&1
signed unsigned.star with key "untrusted" until * -> unsigned.star.sig (glob)
signature check failed for unsigned.star: key "untrusted" is not trusted
[1]
```

`--key-id` overrides the recorded key id.

```scrut
$ starcm sign --expires 30d --key private/untrusted.pem --key-id release-2027 unsigned.star && cut -d" " -f2 unsigned.star.sig
signed unsigned.star with key "release-2027" until * -> unsigned.star.sig (glob)
key=release-2027
```

### verify_all()

`key_id` is the key that signed the file.

```scrut
$ starcm --trusted-keys trusted verify.star
signed: key_id=test release=""
```

A file that doesn't verify stops the run, after `verify_all` has checked
every file, listing each one that failed. `tampered.txt` gets a copy of
`payload.txt`'s signature, which doesn't match its content:

```scrut
$ cp payload.txt.sig tampered.txt.sig && starcm --trusted-keys trusted verify_bad.star 2>&1 | tail -3
Error in verify_all: verify_all: 2 of 3 files didn't verify:
  *tampered.txt: key "test": signature does not match (glob)
  .*[/\\]unsigned\.txt: reading signature: open .*[/\\]unsigned\.txt\.sig: (no such file or directory|The system cannot find the file specified\.) (regex)
```

verify_all() needs `--trusted-keys`.

```python
# examples/signing/edge/verify_untrusted_run.star
load("starcm", "here", "verify_all")

verify_all([here("payload.txt")])
```

```scrut
$ starcm verify_untrusted_run.star 2>&1
Traceback (most recent call last):
  verify_untrusted_run.star:3:11: in <toplevel>
Error in verify_all: verify_all: needs trusted keys: run starcm with --trusted-keys
[1]
```

### Bad keys

```scrut
$ starcm --trusted-keys private main.star 2>&1
reading trusted keys: test.pem: not a PEM "PUBLIC KEY"
[1]
```

```scrut
$ starcm sign --expires 30d --key trusted/test.pub.pem main.star 2>&1
sign: trusted/test.pub.pem: not a PEM "PRIVATE KEY"
[1]
```

### sign()

Signs from Starlark, like `starcm sign`. Signatures are deterministic, so
re-signing unchanged content with the same key and a fixed expiry reports
`changed = False`. An expiry given as a duration is counted from now, so it
re-signs every time.

```python
# examples/signing/edge/sign_fn.star
load("starcm", "sign", "here")

def show(r):
    print("%s: changed=%s value=%s" % (r.label, r.changed, r.value))

show(sign(path = here("payload.txt"), key = here("private/untrusted.pem"), key_id = "other", expires = "30d", label = "other key"))
show(sign(here("payload.txt"), key = here("private/test.pem"), expires = "2099-01-01", label = "test key"))
show(sign(here("payload.txt"), key = here("private/test.pem"), expires = "2099-01-01", label = "unchanged"))
show(sign(here("payload.txt"), key = here("private/test.pem"), expires = "2099-01-01", release = "2026.10", label = "release"))
```

```scrut
$ starcm sign_fn.star && cut -d" " -f1,2,4 payload.txt.sig
other key: changed=True value=other
test key: changed=True value=test
unchanged: changed=False value=test
release: changed=True value=test
ed25519 key=test release=2026.10
```

Deployed runs never sign: `sign()` is disabled under `--trusted-keys`.

```python
# examples/signing/edge/sign_deployed.star
load("starcm", "sign", "here")
sign(here("payload.txt"), key = here("private/test.pem"), expires = "30d", label = "deployed")
```

```scrut
$ starcm sign --expires 30d --key private/test.pem sign_deployed.star >/dev/null && starcm --trusted-keys trusted sign_deployed.star 2>&1
Traceback (most recent call last):
  sign_deployed.star:2:5: in <toplevel>
Error in sign: sign() is disabled when starcm runs with --trusted-keys: signing belongs on the machine that holds the key, not where configuration is deployed
[1]
```

### Expiry

The expiry is part of what's signed, so it can't be extended by editing the
`.sig` file, and an expired signature is refused everywhere a signature is
checked.

```scrut
$ starcm sign --expires 2020-01-01 --key private/test.pem unsigned.star && starcm --trusted-keys trusted unsigned.star 2>&1
signed unsigned.star with key "test" until 2020-01-01T00:00:00Z -> unsigned.star.sig
signature check failed for unsigned.star: key "test": signature expired at 2020-01-01T00:00:00Z (now *) (glob)
[1]
```

```scrut
$ sed 's/expires=2020-01-01T00:00:00Z/expires=2099-01-01T00:00:00Z/' unsigned.star.sig > extended.sig && mv extended.sig unsigned.star.sig && starcm --trusted-keys trusted unsigned.star 2>&1
signature check failed for unsigned.star: key "test": signature does not match
[1]
```

```scrut
$ starcm sign --expires 2020-01-01 --key private/test.pem payload.txt >/dev/null && starcm --trusted-keys trusted verify.star 2>&1 | tail -1
  *payload.txt: key "test": signature expired at 2020-01-01T00:00:00Z (now *) (glob)
```

Signatures in older formats, without an expiry or without the file's
sha256, are refused.

```scrut
$ echo "ed25519 key=test sig=AAAA" > unsigned.star.sig && starcm --trusted-keys trusted unsigned.star 2>&1
signature check failed for unsigned.star: malformed signature file, want "<algorithm> key=<id> expires=<RFC 3339 time> [release=<id>] sha256=<hex> sig=<base64>"
[1]
```

```scrut
$ echo "ed25519 key=test expires=2099-01-01T00:00:00Z sig=AAAA" > unsigned.star.sig && starcm --trusted-keys trusted unsigned.star 2>&1
signature check failed for unsigned.star: malformed signature file, want "<algorithm> key=<id> expires=<RFC 3339 time> [release=<id>] sha256=<hex> sig=<base64>"
[1]
```

Editing the recorded sha256 breaks the signature too:

```scrut
$ starcm sign --expires 2099-01-01 --key private/test.pem unsigned.star >/dev/null && sed 's/sha256=[0-9a-f]*/sha256=0000000000000000000000000000000000000000000000000000000000000000/' unsigned.star.sig > edited.sig && mv edited.sig unsigned.star.sig && starcm --trusted-keys trusted unsigned.star 2>&1
signature check failed for unsigned.star: key "test": signature file records sha256 0000000000000000000000000000000000000000000000000000000000000000, but the signature is for * (glob)
[1]
```

`--expires` is required.

```scrut
$ starcm sign --key private/test.pem unsigned.star 2>&1 | head -1
*Required flag "expires" not set (glob)
```

```scrut
$ starcm sign --expires soon --key private/test.pem unsigned.star 2>&1
sign: invalid expiry "soon": want a duration like 30d, a date like 2026-11-14, or an RFC 3339 time
[1]
```

### --max-expiry

`--max-expiry` also refuses signatures that expire too far from now, so a run
can insist that what it trusts was signed recently.

```scrut
$ starcm sign --expires 30d --key private/test.pem unsigned.star >/dev/null && starcm --trusted-keys trusted --max-expiry 30d unsigned.star
unsigned.star ran
```

```scrut
$ starcm --trusted-keys trusted --max-expiry 7d unsigned.star 2>&1
signature check failed for unsigned.star: key "test": signature expires at *, more than 7d from now (*) (glob)
[1]
```

```scrut
$ starcm --trusted-keys trusted --max-expiry week unsigned.star 2>&1
--max-expiry: invalid duration "week"
[1]
```

```scrut
$ starcm --max-expiry 30d unsigned.star 2>&1
--max-expiry needs --trusted-keys
[1]
```

### --release

A signature without a release is refused by a run that wants one:

```scrut
$ starcm --trusted-keys trusted --release 2026.10 unsigned.star 2>&1
signature check failed for unsigned.star: key "test": signature has no release, want release "2026.10"
[1]
```

The release is signed, so relabelling an old release's signature breaks it:

```scrut
$ starcm sign --expires 30d --key private/test.pem --release 2026.09 unsigned.star >/dev/null && sed 's/release=2026.09/release=2026.10/' unsigned.star.sig > relabelled.sig && mv relabelled.sig unsigned.star.sig && starcm --trusted-keys trusted --release 2026.10 unsigned.star 2>&1
signature check failed for unsigned.star: key "test": signature does not match
[1]
```

Without `--release`, any release is accepted:

```scrut
$ starcm sign --expires 30d --key private/test.pem --release 2026.09 unsigned.star >/dev/null && starcm --trusted-keys trusted unsigned.star
unsigned.star ran
```

verify_all() applies the same check:

```scrut
$ starcm sign --expires 30d --key private/test.pem --release 2026.10 verify.star payload.txt >/dev/null && starcm --trusted-keys trusted --release 2026.10 verify.star | head -1
signed: key_id=test release="2026.10"
```

```scrut
$ starcm sign --expires 30d --key private/test.pem --release 2026.09 payload.txt >/dev/null && starcm --trusted-keys trusted --release 2026.10 verify.star 2>&1 | tail -1
  *payload.txt: key "test": signature is for release "2026.09", want release "2026.10" (glob)
```

```scrut
$ starcm sign --expires 30d --key private/test.pem --release "two words" unsigned.star 2>&1
sign: invalid release "two words": must have no spaces
[1]
```

## See also

- [download](download.md#pinning-by-signature) and [s3_download](s3_download.md) can pin a download to its signature instead of a sha256.
- [verify_all](verify_all.md) to audit files that arrived some other way.
- [--trusted-keys](../cli.md#--trusted-keys-dir).
