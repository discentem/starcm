# Signing

`starcm sign` writes `<file>.sig` next to each file. With `--trusted-keys DIR`
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
$ starcm sign --key private/test.pem main.star lib.star dyn.star verify.star payload.txt
signed main.star with key "test" -> main.star.sig
signed lib.star with key "test" -> lib.star.sig
signed dyn.star with key "test" -> dyn.star.sig
signed verify.star with key "test" -> verify.star.sig
signed payload.txt with key "test" -> payload.txt.sig
```

```scrut
$ cat main.star.sig
ed25519 key=test sig=* (glob)
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
$ starcm sign --key private/untrusted.pem unsigned.star && starcm --trusted-keys trusted unsigned.star 2>&1
signed unsigned.star with key "untrusted" -> unsigned.star.sig
signature check failed for unsigned.star: key "untrusted" is not trusted
[1]
```

`--key-id` overrides the recorded key id.

```scrut
$ starcm sign --key private/untrusted.pem --key-id release-2027 unsigned.star && cut -d" " -f2 unsigned.star.sig
signed unsigned.star with key "release-2027" -> unsigned.star.sig
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
no signature: success=False value=None error=unsigned.txt: reading signature: open unsigned.txt.sig: no such file or directory
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
$ starcm sign --key trusted/test.pub.pem main.star 2>&1
sign: trusted/test.pub.pem: not a PEM "PRIVATE KEY"
[1]
```
