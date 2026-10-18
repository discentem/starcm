# install_mac_pkg

[← All functions](../README.md)

Use `install_mac_pkg` to install a macOS `.pkg`, but only if it's signed by
the certificate your configuration expects. You pin the SHA-256 fingerprint
of the package's installer certificate in the `.star` file. The package is
installed only when all of these hold; otherwise the run stops before
`installer` starts:

- macOS trusts the signature: it's signed by Apple, by a Developer ID
  certificate "issued by Apple for distribution", or by a certificate macOS
  trusts.
- The installer certificate, the first one in the chain, has exactly the
  pinned fingerprint. Pinning an intermediate or root certificate doesn't
  count.
- Apple has notarized it, unless you set `require_notarized = False`.

The package is copied into a private temporary directory first, and the
signature check and `installer` both run on that copy, so the file can't be
swapped between the check and the install. A package from the
[local store](download.md#the-local-store) is hashed again before it's used,
as [exec](exec.md#files-from-the-local-store) does.

`install_mac_pkg` only works on macOS, and installing to `/` needs root, as
`installer` does.

## Syntax

```python
load("starcm", "install_mac_pkg")

install_mac_pkg(
    "agent.pkg",                          # path, required
    signer_sha256     = "1E 77 DE ...",   # required
    require_notarized = True,
    target            = "/",
    id                = "com.example.agent",
    version           = "1.2.0",
    label             = "install agent",  # required
)
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `path` | string | required | The `.pkg` to install, positionally or as `path =`. |
| `signer_sha256` | string | required | SHA-256 fingerprint of the installer certificate. It's accepted as `pkgutil` prints it (`1E 77 DE ...`), with colons, or as plain hex, in any case. |
| `require_notarized` | bool | `True` | Refuse packages Apple hasn't notarized. |
| `target` | string | `"/"` | The volume to install on (`installer -target`). |
| `id`, `version` | string | none | Package id and version. When `pkgutil --pkg-info <id>` already reports this version on `target`, nothing is checked or installed and `changed = False`. Give both or neither. |

`install_mac_pkg` also takes the
[common arguments](../common-functionality.md#common-arguments): `label`
(required), `only_if`, `not_if`, `timeout`, `working_directory` and
`what_if`. With `what_if = True` it checks the signature but doesn't install.

## Finding the fingerprint

Run `pkgutil --check-signature` on a copy of the package you trust, and copy
the first certificate's `SHA256 Fingerprint`:

```bash
pkgutil --check-signature Tailscale-1.98.8-macos.pkg
```

```text
Package "Tailscale-1.98.8-macos.pkg":
   Status: signed by a developer certificate issued by Apple for distribution
   Notarization: trusted by the Apple notary service
   Signed with a trusted timestamp on: 2026-06-24 14:20:40 +0000
   Certificate Chain:
    1. Developer ID Installer: Tailscale Inc. (W5364U7YZB)
       Expires: 2028-12-12 15:02:21 +0000
       SHA256 Fingerprint:
           1E 77 DE 6F E4 7D 42 69 84 C5 AA 18 0D 58 1F 32 46 19 7F 73 C0 C7
           13 27 7B 84 40 08 B9 36 E0 93
       ------------------------------------------------------------------------
    2. Developer ID Certification Authority
    ...
```

The installer certificate changes when the vendor renews it, about every five
years. A package signed with a renewed certificate is refused until you pin
the new fingerprint.

## Returns

A [starcm_result](../common-functionality.md#starcm_result):

| Field | Value |
| --- | --- |
| `success` | `True` |
| `changed` | `True` when the package was installed (or, with `what_if`, would be); `False` when `id` / `version` was already installed |
| `message` | `installed <path>, signed by "<certificate>"`, `would install ...` or `<id> <version> is already installed` |
| `value` | A `pkg_signature` struct with `signer` (the certificate's name), `signer_sha256`, `team_id` and `notarized`. It's `None` when nothing was checked because the version was already installed. |

A signature that fails any check, a missing file, or a failing `installer`
stops the run.

## Examples

The examples that install need a real signed package and root, so they're
shown here but not run in CI. The refusals at the end are run, on Linux,
macOS and Windows. On Linux and Windows the same calls fail because
`install_mac_pkg` only works on macOS.

### Download a package and install it

`download` without `save_to` puts the package in the local store, and
`install_mac_pkg` re-hashes it there before checking its signature. With `id`
and `version`, later runs do nothing once 1.2.0 is installed.

<!-- fixture: ../fixtures/install_mac_pkg/install.star -->
```python
load("starcm", "download", "install_mac_pkg")

# From `pkgutil --check-signature` on a copy of the package you trust.
AGENT_SIGNER = "1E 77 DE 6F E4 7D 42 69 84 C5 AA 18 0D 58 1F 32 46 19 7F 73 C0 C7 13 27 7B 84 40 08 B9 36 E0 93"
AGENT_SHA256 = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08"

pkg = download(
    label  = "fetch agent",
    url    = "https://downloads.example.com/agent-1.2.0.pkg",
    sha256 = AGENT_SHA256,
)

r = install_mac_pkg(
    pkg.value,
    signer_sha256 = AGENT_SIGNER,
    id            = "com.example.agent",
    version       = "1.2.0",
    label         = "install agent",
)
print("%s (changed=%s)" % (r.message, r.changed))
```

### Check a package without installing it

<!-- fixture: ../fixtures/install_mac_pkg/preview.star -->
```python
load("starcm", "install_mac_pkg", "vars")

r = install_mac_pkg(
    vars["pkg"],
    signer_sha256 = vars["signer"],
    what_if       = True,
    label         = "check " + vars["pkg"],
)
print(r.message)
print("team %s, notarized=%s" % (r.value.team_id, r.value.notarized))
```

```bash
starcm --var pkg=Tailscale-1.98.8-macos.pkg --var "signer=1E 77 DE 6F E4 7D 42 69 84 C5 AA 18 0D 58 1F 32 46 19 7F 73 C0 C7 13 27 7B 84 40 08 B9 36 E0 93" preview.star
```

```text
would install Tailscale-1.98.8-macos.pkg, signed by "Developer ID Installer: Tailscale Inc. (W5364U7YZB)"
team W5364U7YZB, notarized=True
```

### Unsigned packages are refused

These examples run in a scratch copy of
[docs/fixtures/install_mac_pkg/](../fixtures/install_mac_pkg). On macOS,
`pkgbuild` makes an unsigned package; elsewhere any file will do.

```scrut
$ cp -R "$TESTDIR/../fixtures/install_mac_pkg/." "$TMPDIR" && cd "$TMPDIR" && if command -v pkgbuild >/dev/null; then pkgbuild --nopayload --identifier com.example.unsigned --version 1.0 unsigned.pkg >/dev/null 2>&1; else echo not a pkg > unsigned.pkg; fi
```

<!-- fixture: ../fixtures/install_mac_pkg/unsigned.star -->
```python
load("starcm", "install_mac_pkg")

install_mac_pkg(
    "unsigned.pkg",
    signer_sha256 = "1E 77 DE 6F E4 7D 42 69 84 C5 AA 18 0D 58 1F 32 46 19 7F 73 C0 C7 13 27 7B 84 40 08 B9 36 E0 93",
    label         = "unsigned",
)
print("never reached")
```

```scrut
$ starcm unsigned.star 2>&1
Traceback (most recent call last):
  unsigned.star:3:16: in <toplevel>
Error in install_mac_pkg: install_mac_pkg\(label="unsigned"\): (unsigned\.pkg: signature is not trusted: no signature|install_mac_pkg\(\) only works on macOS) (regex)
[1]
```

### A Team ID isn't enough

<!-- fixture: ../fixtures/install_mac_pkg/bad_fingerprint.star -->
```python
load("starcm", "install_mac_pkg")

# A Team ID is not enough: the installer certificate's sha256 is required.
install_mac_pkg("agent.pkg", signer_sha256 = "W5364U7YZB", label = "team id")
```

```scrut
$ starcm bad_fingerprint.star 2>&1
Traceback (most recent call last):
  bad_fingerprint.star:4:16: in <toplevel>
Error in install_mac_pkg: install_mac_pkg(label="team id"): signer_sha256: "W5364U7YZB" is not a sha256 fingerprint (64 hex digits)
[1]
```

## See also

- [download](download.md) to fetch the package first.
- [verify](verify.md) to check files signed with `starcm sign` instead.
- [tests/install_mac_pkg.md](../../tests/install_mac_pkg.md).
