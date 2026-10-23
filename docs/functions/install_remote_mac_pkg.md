# install_remote_mac_pkg

[← All functions](../README.md)

Use `install_remote_mac_pkg` to fetch a macOS `.pkg` from a web server or an
S3 bucket and install it in one step. It does for a package what
[load_remote](load_remote.md) does for a `.star` file: the package always
goes into the [local store](download.md#the-local-store), in a folder named
after its sha256, and is installed from there with every check
[install_mac_pkg](install_mac_pkg.md) makes. So the package installed is the
one that matched its pin, and a second run doesn't fetch it again.

Pin the package by `sha256` or, under
[--trusted-keys](../cli.md#--trusted-keys-dir), by `signature`, as for
[download](download.md#pinning-by-signature). That pin says which file to
fetch. `signer` then says which developer must have signed it, through
Apple. Both must hold.

With `id` and `version`, a run where that version is already installed
fetches nothing.

`install_remote_mac_pkg` only works on macOS, and installing to `/` needs
root, as `installer` does.

## Syntax

```python
load("starcm", "install_remote_mac_pkg")

install_remote_mac_pkg(
    "https://releases.example.com/agent-1.2.0.pkg",                # url, required
    signature         = "https://releases.example.com/agent-1.2.0.pkg.sig",  # or sha256 = "<hex>"
    signer            = "Developer ID Installer: Example Corp (ABCDE12345)",  # required
    require_notarized = True,
    target            = "/",
    id                = "com.example.agent",
    version           = "1.2.0",
    label             = "install agent",                           # required
)
install_remote_mac_pkg(
    url       = "s3://releases/agent-1.2.0.pkg",
    signature = "s3://releases/agent-1.2.0.pkg.sig",
    signer    = "Developer ID Installer: Example Corp (ABCDE12345)",
    endpoint  = vars["s3_endpoint"],  # optional, for S3-compatible stores
    label     = "install agent",
)
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `url` | string | required | The package, positionally or as `url =`: an `http://` or `https://` URL, or `s3://bucket/key`. Its name must end in `.pkg`. |
| `sha256` | string | none | The sha256 the package must have. |
| `signature` | string | none | The URL of the package's signature, checked against `--trusted-keys`. For `s3://` URLs, an `s3://` URL in the same bucket. See [pinning by signature](download.md#pinning-by-signature). |
| `signer` | string | required | The installer certificate's name, as for [install_mac_pkg](install_mac_pkg.md#finding-the-signer). |
| `require_notarized` | bool | `True` | Refuse packages Apple hasn't notarized. `None` counts as leaving it out. |
| `target` | string | `"/"` | The volume to install on, as an absolute path. |
| `id`, `version` | string | none | Package id and version. When `pkgutil --pkg-info <id>` already reports this version on `target`, nothing is fetched, checked or installed and `changed = False`. Give both or neither. |
| `endpoint` | string | AWS | `s3://` only: the S3-compatible server, as for [s3_download](s3_download.md). |
| `region` | string | AWS configuration, else `us-east-1` | `s3://` only: the bucket's region. |

Give exactly one of `sha256` and `signature`. S3 credentials come from the
AWS SDK's usual sources, as for [s3_download](s3_download.md).

`install_remote_mac_pkg` also takes the
[common arguments](../common-functionality.md#common-arguments): `label`
(required), `only_if`, `not_if`, `timeout` and `what_if`. With
`what_if = True` it fetches the package into the store and checks its
signature but doesn't install it.

## Returns

A [starcm_result](../common-functionality.md#starcm_result), as for
[install_mac_pkg](install_mac_pkg.md#returns):

| Field | Value |
| --- | --- |
| `success` | `True` |
| `changed` | `True` when the package was installed (or, with `what_if`, would be); `False` when `id` / `version` was already installed |
| `message` | `installed <url>, signed by "<signer>"`, `would install ...` or `<id> <version> is already installed` |
| `value` | A `pkg_signature` struct with `signer`, `team_id`, `signer_sha256` and `notarized`, or `None` when the version was already installed. |

A package that doesn't match its pin, a refused signature, a failed download
or a failing `installer` stops the run.

## Examples

Installing needs a real signed package and root, so the first example is
shown here but not run in CI. The refusals after it are run, on Linux, macOS
and Windows. On Linux and Windows, a call that gets past its argument checks
fails because `install_remote_mac_pkg` only works on macOS, before anything
is fetched.

### Install a signed release

The release's signature, made with `starcm sign`, is served next to it, and
`--trusted-keys` must trust the key that made it. With `id` and `version`,
later runs do nothing once 1.2.0 is installed.

```python
# examples/install_remote_mac_pkg/install.star
load("starcm", "install_remote_mac_pkg")

r = install_remote_mac_pkg(
    "https://releases.example.com/agent-1.2.0.pkg",
    # No sha256 here: the signature, checked against --trusted-keys, says
    # which sha256 the package must have.
    signature = "https://releases.example.com/agent-1.2.0.pkg.sig",
    # The installer certificate's name, from `pkgutil --check-signature`.
    signer    = "Developer ID Installer: Example Corp (ABCDE12345)",
    id        = "com.example.agent",
    version   = "1.2.0",
    label     = "install agent",
)
print("%s (changed=%s)" % (r.message, r.changed))
```

```bash
starcm --trusted-keys /etc/starcm/trusted install.star
```

```text
installed https://releases.example.com/agent-1.2.0.pkg, signed by "Developer ID Installer: Example Corp (ABCDE12345)" (changed=True)
```

### Unsigned packages are refused

These examples run in a scratch copy of
[examples/install_remote_mac_pkg/](../../examples/install_remote_mac_pkg).
The package they fetch,
[remote/unsigned.pkg](../../examples/install_remote_mac_pkg/remote/unsigned.pkg),
was made with `pkgbuild` and not signed. It's served from this repository by
`python3 -m http.server 8000`. `$store` is a scratch store (on Windows, under
`%LocalAppData%`, since starcm refuses a store on a drive other accounts can
change).

```scrut
$ cp -R "$TESTDIR/../../examples/install_remote_mac_pkg/." "$TMPDIR" && cd "$TMPDIR"
```

```scrut
$ store="$TMPDIR/store"; case "$(uname -s)" in MINGW*|MSYS*|CYGWIN*) store_root=$(mktemp -d "$(cygpath -u "$LOCALAPPDATA")/starcm-docs.XXXXXX") && store="$store_root/store" ;; esac
```

The package matches its sha256, so on macOS it's fetched into the store, and
then refused because no developer signed it:

```python
# examples/install_remote_mac_pkg/unsigned.star
load("starcm", "install_remote_mac_pkg")

install_remote_mac_pkg(
    "http://localhost:8000/examples/install_remote_mac_pkg/remote/unsigned.pkg",
    sha256 = "4facc51893659121661fc6337f2b63ce5abf64b24ba7a73f399d656eb6d71edf",
    signer = "Developer ID Installer: Example Corp (ABCDE12345)",
    label  = "unsigned",
)
print("never reached")
```

```scrut
$ starcm --store-dir "$store" unsigned.star 2>&1
Traceback (most recent call last):
  unsigned.star:3:23: in <toplevel>
Error in install_remote_mac_pkg: install_remote_mac_pkg\(label="unsigned"\): (http://localhost:8000/examples/install_remote_mac_pkg/remote/unsigned\.pkg: not signed with a certificate Apple issued for distribution: no signature|install_remote_mac_pkg\(\) only works on macOS) (regex)
[1]
```

## Errors and edge cases

These are here mostly as tests: CI runs them to check error messages and
behaviour that the examples above don't show. Arguments are checked before
anything else, so they fail the same way on every platform. They run in the
copy of
[examples/install_remote_mac_pkg/edge/](../../examples/install_remote_mac_pkg/edge).

```scrut
$ cd edge
```

### A pin is required

```python
# examples/install_remote_mac_pkg/edge/no_pin.star
load("starcm", "install_remote_mac_pkg")

install_remote_mac_pkg(
    "http://localhost:8000/examples/install_remote_mac_pkg/remote/unsigned.pkg",
    signer = "Developer ID Installer: Example Corp (ABCDE12345)",
    label  = "no pin",
)
```

```scrut
$ starcm --store-dir "$store" no_pin.star 2>&1
Traceback (most recent call last):
  no_pin.star:3:23: in <toplevel>
Error in install_remote_mac_pkg: install_remote_mac_pkg(label="no pin"): pass exactly one of sha256 or signature
[1]
```

### URLs

```python
# examples/install_remote_mac_pkg/edge/not_pkg.star
load("starcm", "install_remote_mac_pkg")

install_remote_mac_pkg(
    "https://releases.example.com/agent-1.2.0.dmg",
    sha256 = "4facc51893659121661fc6337f2b63ce5abf64b24ba7a73f399d656eb6d71edf",
    signer = "Developer ID Installer: Example Corp (ABCDE12345)",
    label  = "not a .pkg",
)
```

```scrut
$ starcm --store-dir "$store" not_pkg.star 2>&1
Traceback (most recent call last):
  not_pkg.star:3:23: in <toplevel>
Error in install_remote_mac_pkg: install_remote_mac_pkg(label="not a .pkg"): https://releases.example.com/agent-1.2.0.dmg doesn't name a .pkg file
[1]
```

An S3 package's signature must be in the same bucket:

```python
# examples/install_remote_mac_pkg/edge/other_bucket.star
load("starcm", "install_remote_mac_pkg")

install_remote_mac_pkg(
    "s3://releases/agent-1.2.0.pkg",
    signature = "s3://elsewhere/agent-1.2.0.pkg.sig",
    signer    = "Developer ID Installer: Example Corp (ABCDE12345)",
    label     = "other bucket",
)
```

```scrut
$ starcm --store-dir "$store" other_bucket.star 2>&1
Traceback (most recent call last):
  other_bucket.star:3:23: in <toplevel>
Error in install_remote_mac_pkg: install_remote_mac_pkg(label="other bucket"): signature s3://elsewhere/agent-1.2.0.pkg.sig must be an s3:// URL in bucket "releases"
[1]
```

```scrut
$ rm -rf "$store" ${store_root:+"$store_root"}
```

## See also

- [install_mac_pkg](install_mac_pkg.md) for a package that's already on the machine.
- [load_remote](load_remote.md), the same for `.star` files.
- [download](download.md#pinning-by-signature) for how pins and signatures work.
- [inspect](inspect.md) reports each `install_remote_mac_pkg` call's URL, or `null` when it's computed.
