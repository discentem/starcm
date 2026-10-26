# verify_all

[← All functions](../README.md)

Use `verify_all` to audit files that arrived some other way, for example
through MDM or a file share, against signatures made with [sign](sign.md) by
one of the keys passed with [--trusted-keys](../cli.md#--trusted-keys-dir).
It checks every file before deciding, so a run that fails lists every file
that didn't verify and why, not just the first.

`verify_all` is for reporting. To verify something you're about to use,
let the function that uses it check the exact bytes it uses, so the file
can't be swapped between the check and its use:

- [download](download.md#pinning-by-signature) and
  [s3_download](s3_download.md) pin a download to its signature (or its
  sha256), and the [local store](download.md#the-local-store) hashes the file
  again before [exec](exec.md#files-from-the-local-store), [file](file.md)
  or [extract](extract.md) use it.
- [load_remote](load_remote.md) pins a remote `.star` file the same way
  before loading it.
- [install_mac_pkg](install_mac_pkg.md) and
  [install_remote_mac_pkg](install_remote_mac_pkg.md) check a package's Apple
  signature on the copy they install, and
  [file](file.md#copy-a-signed-binary-into-place) with `source` and `signer`
  does the same for binaries.

`verify_all` only works when starcm runs with `--trusted-keys`, which also
means the `.star` file calling it must itself be signed. It applies the same
expiry and release checks, including
[--max-expiry](../cli.md#--max-expiry-duration) and
[--release](../cli.md#--release-id), as the checks on `.star` files.

## Syntax

```python
load("starcm", "here", "path", "verify_all")

verify_all(["/opt/agent/agent.pkg", "/opt/agent/plugin.pkg"])
verify_all(path.glob(here("payloads/*.pkg")))
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `paths` | list of strings | required | The files to check, as absolute paths (`~` expands), positionally or as `paths =`. Each one's signature is read from next to it (`<path>.sig`). Pass the result of [path.glob](path.md#pathglob) or [path.list](path.md#pathlist) to check a directory. |

`verify_all` only reads, so it's a plain function: it takes no `label` or
other [common arguments](../common-functionality.md#common-arguments) and
returns its answer directly, not a `starcm_result`. It runs under `what_if`
too.

## Returns

A list with one `verification` struct per file, in the order given:

| Field | Description |
| --- | --- |
| `path` | The file checked, as an absolute path |
| `key_id` | The id of the key that signed it |
| `release` | The release it was signed for, or `""` |
| `expires` | When the signature expires, in RFC 3339 |

If any file doesn't verify, it stops the run instead, after checking all of
them, with one line per failing file:

```text
verify_all: 2 of 5 files didn't verify:
  /opt/agent/plugin.pkg: key "release": signature does not match
  /opt/agent/extra.pkg: reading signature: open /opt/agent/extra.pkg.sig: no such file or directory
```

An empty list returns `[]`. Running without `--trusted-keys` stops the run.

## Examples

The examples use the test-only key from [docs/keys/](../keys). `check.star`
and `agent.pkg` are signed first. `unsigned.pkg` isn't, and `other.pkg` is
signed and then changed.

```scrut
$ cp -R "$TESTDIR/../../examples/verify_all/." "$TMPDIR" && cd "$TMPDIR" && mkdir trusted && cp "$TESTDIR/../keys/private/test.pem" release.pem && cp "$TESTDIR/../keys/trusted/test.pub.pem" trusted/release.pub.pem
```

```scrut
$ starcm sign --expires 2099-01-01 --key release.pem agent.pkg other.pkg check.star && echo "malware" >> other.pkg
signed agent.pkg with key "release" until 2099-01-01T00:00:00Z -> agent.pkg.sig
signed other.pkg with key "release" until 2099-01-01T00:00:00Z -> other.pkg.sig
signed check.star with key "release" until 2099-01-01T00:00:00Z -> check.star.sig
```

### Check many files

```python
# examples/verify_all/check.star
load("starcm", "here", "verify_all")

# Checks every file first, then stops the run once, listing each one that
# didn't verify.
for v in verify_all([here("agent.pkg"), here("unsigned.pkg"), here("other.pkg")]):
    print("%s: key_id=%s" % (v.path, v.key_id))
```

```scrut
$ starcm --trusted-keys trusted check.star 2>&1
Traceback (most recent call last):
  check.star:5:20: in <toplevel>
Error in verify_all: verify_all: 2 of 3 files didn't verify:
  .*[/\\]unsigned\.pkg: reading signature: open .*[/\\]unsigned\.pkg\.sig: (no such file or directory|The system cannot find the file specified\.) (regex)
  *other.pkg: key "release": signature does not match (glob)
[1]
```

With every file signed, it returns their verifications:

```scrut
$ starcm sign --expires 2099-01-01 --key release.pem unsigned.pkg other.pkg >/dev/null && starcm --trusted-keys trusted check.star
*agent.pkg: key_id=release (glob)
*unsigned.pkg: key_id=release (glob)
*other.pkg: key_id=release (glob)
```

An expired signature fails the same way:

```scrut
$ starcm sign --expires 2020-01-01 --key release.pem agent.pkg >/dev/null && starcm --trusted-keys trusted check.star 2>&1 | tail -2
Error in verify_all: verify_all: 1 of 3 files didn't verify:
  *agent.pkg: key "release": signature expired at 2020-01-01T00:00:00Z (now *) (glob)
```

## Errors and edge cases

These are here mostly as tests: CI runs them to check error messages and
behaviour that the examples above don't show.

### verify_all needs --trusted-keys

```scrut
$ starcm check.star 2>&1
Traceback (most recent call last):
  check.star:5:20: in <toplevel>
Error in verify_all: verify_all: needs trusted keys: run starcm with --trusted-keys
[1]
```

## See also

- [sign](sign.md) for keys and signing.
- [download](download.md#pinning-by-signature) and [load_remote](load_remote.md) to verify what you fetch.
