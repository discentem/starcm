# verify

[← All functions](../README.md)

Use `verify` to check a file, such as a downloaded installer, against a
signature made with [sign](sign.md) by one of the keys passed with
[--trusted-keys](../cli.md#--trusted-keys-dir). A file that doesn't verify
is an answer, not an error: `verified` is `False`, so your configuration
decides what happens next, usually `fail()`.

`verify` only works when starcm runs with `--trusted-keys`, which also means
the `.star` file calling it must itself be signed. It applies the same
expiry and release checks, including
[--max-expiry](../cli.md#--max-expiry-duration) and
[--release](../cli.md#--release-id), as the checks on `.star` files.

## Syntax

```python
load("starcm", "verify")

verify("/opt/agent/agent.pkg")   # reads agent.pkg.sig next to it
verify(path = "/opt/agent/agent.pkg", signature = "/opt/agent/sigs/agent.pkg.sig")
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `path` | string | required | The file to check, positionally or as `path =`. An absolute path; a leading `~` expands to the home directory. |
| `signature` | string | `path + ".sig"` | The signature file, as an absolute path. |

`verify` only reads, so it's a plain function: it takes no `label` or other
[common arguments](../common-functionality.md#common-arguments) and returns
its answer directly, not a `starcm_result`. It runs under `what_if` too.

## Returns

A `verification` struct:

| Field | Verified | Not verified |
| --- | --- | --- |
| `verified` | `True` | `False` |
| `key_id` | The id of the key that signed it | `None` |
| `release` | The release it was signed for, or `""` | `None` |
| `expires` | When the signature expires, in RFC 3339 | `None` |
| `error` | `None` | Why, e.g. `<path>: key "release": signature does not match` or `... signature expired at <expiry> (now <time>)` |

Running without `--trusted-keys` stops the run.

## Examples

The examples use the test-only key from
[docs/keys/](../keys). Every `.star` file
and `agent.pkg` are signed first; `unsigned.pkg` isn't.

```scrut
$ cp -R "$TESTDIR/../../examples/verify/." "$TMPDIR" && cd "$TMPDIR" && mkdir trusted && cp "$TESTDIR/../keys/private/test.pem" release.pem && cp "$TESTDIR/../keys/trusted/test.pub.pem" trusted/release.pub.pem
```

```scrut
$ starcm sign --expires 2099-01-01 --key release.pem agent.pkg check.star must_verify.star detached.star
signed agent.pkg with key "release" until 2099-01-01T00:00:00Z -> agent.pkg.sig
signed check.star with key "release" until 2099-01-01T00:00:00Z -> check.star.sig
signed must_verify.star with key "release" until 2099-01-01T00:00:00Z -> must_verify.star.sig
signed detached.star with key "release" until 2099-01-01T00:00:00Z -> detached.star.sig
```

### Check files

```python
# examples/verify/check.star
load("starcm", "here", "verify")

for f in ["agent.pkg", "unsigned.pkg"]:
    v = verify(here(f))
    print("%s: verified=%s key_id=%s" % (f, v.verified, v.key_id))
    if not v.verified:
        print("  error: %s" % v.error)
```

```scrut
$ starcm --trusted-keys trusted check.star
agent.pkg: verified=True key_id=release
unsigned.pkg: verified=False key_id=None
  error: .*[/\\]unsigned\.pkg: reading signature: open .*[/\\]unsigned\.pkg\.sig: (no such file or directory|The system cannot find the file specified\.) (regex)
```

### Refuse to install a tampered file

```python
# examples/verify/must_verify.star
load("starcm", "exec", "verify", "here")

check = verify(here("agent.pkg"))
if not check.verified:
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
Error in fail: fail: *agent.pkg: key "release": signature does not match (glob)
[1]
```

An expired signature fails the same way:

```scrut
$ mv agent.orig agent.pkg && starcm sign --expires 2020-01-01 --key release.pem agent.pkg >/dev/null && starcm --trusted-keys trusted must_verify.star 2>&1
Traceback (most recent call last):
  must_verify.star:5:9: in <toplevel>
Error in fail: fail: *agent.pkg: key "release": signature expired at 2020-01-01T00:00:00Z (now *) (glob)
[1]
```

### Keep signatures somewhere else

```python
# examples/verify/detached.star
load("starcm", "verify", "here")

v = verify(here("agent.pkg"), signature = here("sigs/agent.pkg.sig"))
print("verified=%s key_id=%s expires=%s" % (v.verified, v.key_id, v.expires))
```

```scrut
$ starcm sign --expires 2099-01-01 --key release.pem agent.pkg >/dev/null && mkdir sigs && mv agent.pkg.sig sigs/ && starcm --trusted-keys trusted detached.star
verified=True key_id=release expires=2099-01-01T00:00:00Z
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
