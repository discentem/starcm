# verify

[← All functions](../README.md)

Use `verify` to check a file, such as a downloaded installer, against a
signature made with [sign](sign.md) by one of the keys passed with
[--trusted-keys](../cli.md#--trusted-keys-dir). A file that doesn't verify
is a failed result, not an error, so your configuration decides what happens
next, usually `fail()`.

`verify` only works when starcm runs with `--trusted-keys`, which also means
the `.star` file calling it must itself be signed. It applies the same
expiry checks, including [--max-expiry](../cli.md#--max-expiry-duration),
as the checks on `.star` files.

## Syntax

```python
load("starcm", "verify")

verify("agent.pkg", label = "check agent")                       # reads agent.pkg.sig
verify(path = "agent.pkg", signature = "sigs/agent.pkg.sig", label = "check agent")
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `path` | string | required | The file to check, positionally or as `path =`. |
| `signature` | string | `path + ".sig"` | The signature file. |

`verify` also takes the [common arguments](../common-functionality.md#common-arguments):
`label` (required), `only_if`, `not_if`, `timeout`, `working_directory` and
`what_if`. Only `only_if` and `not_if` affect it.

## Returns

A [starcm_result](../common-functionality.md#starcm_result):

| Field | Verified | Not verified |
| --- | --- | --- |
| `success` | `True` | `False` |
| `value` | The id of the key that signed it | `None` |
| `message` | `<path> verified with key "<id>", signature expires <expiry>` | `""` |
| `error` | `"<nil>"` | Why, e.g. `<path>: key "release": signature does not match` or `... signature expired at <expiry> (now <time>)` |
| `changed` | `False` | `False` |

Running without `--trusted-keys` stops the run.

## Examples

The examples use the test-only key from
[tests/fixtures/signing/](../../tests/fixtures/signing). Every `.star` file
and `agent.pkg` are signed first; `unsigned.pkg` isn't.

```scrut
$ cp -R "$TESTDIR/../fixtures/verify/." "$TMPDIR" && cd "$TMPDIR" && mkdir trusted && cp "$TESTDIR/../../tests/fixtures/signing/private/test.pem" release.pem && cp "$TESTDIR/../../tests/fixtures/signing/trusted/test.pub.pem" trusted/release.pub.pem
```

```scrut
$ starcm sign --expires 2099-01-01 --key release.pem agent.pkg check.star must_verify.star detached.star
signed agent.pkg with key "release" until 2099-01-01T00:00:00Z -> agent.pkg.sig
signed check.star with key "release" until 2099-01-01T00:00:00Z -> check.star.sig
signed must_verify.star with key "release" until 2099-01-01T00:00:00Z -> must_verify.star.sig
signed detached.star with key "release" until 2099-01-01T00:00:00Z -> detached.star.sig
```

### Check files

```scrut
$ cat check.star
load("starcm", "verify")

for f in ["agent.pkg", "unsigned.pkg"]:
    r = verify(f, label = "check " + f)
    print("%s: success=%s key=%s" % (f, r.success, r.value))
    if not r.success:
        print("  error: %s" % r.error)
```

```scrut
$ starcm --trusted-keys trusted check.star
agent.pkg: success=True key=release
unsigned.pkg: success=False key=None
  error: unsigned\.pkg: reading signature: open unsigned\.pkg\.sig: (no such file or directory|The system cannot find the file specified\.) (regex)
```

### Refuse to install a tampered file

```scrut
$ cat must_verify.star
load("starcm", "exec", "verify")

check = verify("agent.pkg", label = "check agent signature")
if not check.success:
    fail(check.error)
exec(label = "install agent", cmd = "cat", args = ["agent.pkg"], live_output = True)
```

```scrut
$ starcm --trusted-keys trusted must_verify.star
agent payload
```

```scrut
$ cp agent.pkg agent.orig && echo "malware" >> agent.pkg && starcm --trusted-keys trusted must_verify.star 2>&1
Traceback (most recent call last):
  must_verify.star:5:9: in <toplevel>
Error in fail: fail: agent.pkg: key "release": signature does not match
[1]
```

An expired signature fails the same way:

```scrut
$ mv agent.orig agent.pkg && starcm sign --expires 2020-01-01 --key release.pem agent.pkg >/dev/null && starcm --trusted-keys trusted must_verify.star 2>&1
Traceback (most recent call last):
  must_verify.star:5:9: in <toplevel>
Error in fail: fail: agent.pkg: key "release": signature expired at 2020-01-01T00:00:00Z (now *) (glob)
[1]
```

### Keep signatures somewhere else

```scrut
$ cat detached.star
load("starcm", "verify")

r = verify("agent.pkg", signature = "sigs/agent.pkg.sig", label = "check with a detached signature")
print("success=%s key=%s" % (r.success, r.value))
```

```scrut
$ starcm sign --expires 2099-01-01 --key release.pem agent.pkg >/dev/null && mkdir sigs && mv agent.pkg.sig sigs/ && starcm --trusted-keys trusted detached.star
success=True key=release
```

### `verify` needs `--trusted-keys`

```scrut
$ starcm check.star 2>&1
Traceback (most recent call last):
  check.star:4:15: in <toplevel>
Error in verify: verify() needs trusted keys: run starcm with --trusted-keys
[1]
```

## See also

- [sign](sign.md) for keys and signing.
- [download](download.md) to fetch the file first.
- [tests/signing.md](../../tests/signing.md).
