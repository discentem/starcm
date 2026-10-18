# Signing

`starcm sign` writes `<file>.sig` next to each file. Every signature expires:
`--expires` takes a duration from now (`30d`, `36h`), a date (`2026-11-14`,
midnight UTC) or an RFC 3339 time. With `--trusted-keys DIR`
(a directory of `*.pem` public keys, named by key id), every `.star` file must
have a valid signature before it runs, including files pulled in by `load()`
and `load_dynamic()`, and `verify()` checks other files.

The keys in `fixtures/signing/private/` are test-only, made with
`openssl genpkey -algorithm ed25519`.

```scrut
$ cp -R "$TESTDIR/fixtures/signing/." "$TMPDIR" && cd "$TMPDIR"
```

## Signing

The key id defaults to the key file's name before the first dot.

```scrut
$ starcm sign --expires 30d --key private/test.pem main.star lib.star dyn.star verify.star payload.txt
signed main.star with key "test" until * -> main.star.sig (glob)
signed lib.star with key "test" until * -> lib.star.sig (glob)
signed dyn.star with key "test" until * -> dyn.star.sig (glob)
signed verify.star with key "test" until * -> verify.star.sig (glob)
signed payload.txt with key "test" until * -> payload.txt.sig (glob)
```

```scrut
$ cat main.star.sig
ed25519 key=test expires=* sha256=* sig=* (glob)
```

## Signed files run

`main.star` loads `lib.star` with `load()` and `dyn.star` with `load_dynamic()`.

```scrut
$ starcm --trusted-keys trusted main.star
hello, nobody
dyn.star ran
```

## Without --trusted-keys nothing is checked

```scrut
$ starcm unsigned.star
unsigned.star ran
```

## Unsigned files are refused

```scrut
$ starcm --trusted-keys trusted unsigned.star 2>&1
signature check failed for unsigned.star: no signature file unsigned.star.sig
[1]
```

## Edited files are refused, whichever way they're loaded

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
Error in load_dynamic: signature check failed for dyn.star: key "test": signature does not match
[1]
```

```scrut
$ mv dyn.orig dyn.star && starcm --trusted-keys trusted main.star
hello, nobody
dyn.star ran
```

## Signatures from keys that aren't trusted are refused

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

## verify()

A file that doesn't verify is a failed result, not an error, so scripts decide
whether to `fail()`. `value` is the key id that signed it.

```scrut
$ starcm --trusted-keys trusted verify.star
signed: success=True value=test error=<nil>
signature= kwarg: success=True value=test error=<nil>
tampered: success=False value=None error=tampered.txt: key "test": signature does not match
no signature: success=False value=None error=unsigned\.txt: reading signature: open unsigned\.txt\.sig: (no such file or directory|The system cannot find the file specified\.) (regex)
```

verify() needs `--trusted-keys`.

```scrut
$ starcm verify_untrusted_run.star 2>&1
Traceback (most recent call last):
  verify_untrusted_run.star:3:7: in <toplevel>
Error in verify: verify() needs trusted keys: run starcm with --trusted-keys
[1]
```

## Bad keys

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

## sign()

Signs from Starlark, like `starcm sign`. Signatures are deterministic, so
re-signing unchanged content with the same key and a fixed expiry reports
`changed = False`. An expiry given as a duration is counted from now, so it
re-signs every time.

```scrut
$ starcm sign_fn.star && cut -d" " -f1,2 payload.txt.sig
other key: changed=True value=other
test key: changed=True value=test
unchanged: changed=False value=test
ed25519 key=test
```

Deployed runs never sign: `sign()` is disabled under `--trusted-keys`.

```scrut
$ starcm sign --expires 30d --key private/test.pem sign_deployed.star >/dev/null && starcm --trusted-keys trusted sign_deployed.star 2>&1
Traceback (most recent call last):
  sign_deployed.star:2:5: in <toplevel>
Error in sign: sign() is disabled when starcm runs with --trusted-keys: signing belongs on the machine that holds the key, not where configuration is deployed
[1]
```

## Expiry

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
$ starcm sign --expires 2020-01-01 --key private/test.pem payload.txt >/dev/null && starcm --trusted-keys trusted verify.star | head -1
signed: success=False value=None error=payload.txt: key "test": signature expired at 2020-01-01T00:00:00Z (now *) (glob)
```

Signatures in older formats, without an expiry or without the file's
sha256, are refused.

```scrut
$ echo "ed25519 key=test sig=AAAA" > unsigned.star.sig && starcm --trusted-keys trusted unsigned.star 2>&1
signature check failed for unsigned.star: malformed signature file, want "<algorithm> key=<id> expires=<RFC 3339 time> sha256=<hex> sig=<base64>"
[1]
```

```scrut
$ echo "ed25519 key=test expires=2099-01-01T00:00:00Z sig=AAAA" > unsigned.star.sig && starcm --trusted-keys trusted unsigned.star 2>&1
signature check failed for unsigned.star: malformed signature file, want "<algorithm> key=<id> expires=<RFC 3339 time> sha256=<hex> sig=<base64>"
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

## --max-expiry

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
