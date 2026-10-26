# install_mac_pkg

[← All functions](../README.md)

Use `install_mac_pkg` to install a macOS `.pkg`, but only if it's signed by
the developer your configuration expects. You name that developer's
installer certificate in the `.star` file, for example
`"Developer ID Installer: Tailscale Inc. (W5364U7YZB)"`. The package is
installed only when all of these hold; otherwise the run stops before
`installer` starts:

- The certificate was issued by Apple for distribution: a Developer ID
  certificate, or Apple's own software signing. Apple only issues a Developer
  ID certificate to the team its name and Team ID belong to, so the name can
  be trusted. Certificates from any other authority are refused, even ones
  macOS trusts, because they could carry any name. So are unsigned packages,
  and untrusted, expired or development certificates.
- The certificate chain ends at Apple's root certificate.
- The installer certificate, the first one in the chain, has exactly the
  name you gave, including the Team ID in parentheses. Naming an
  intermediate or root certificate doesn't count.
- Apple has notarized it, unless you set `require_notarized = False`.
- If you pin `sha256`, the file's sha256 matches it.

Because the developer is named rather than one certificate, packages signed
with the developer's renewed certificate keep working.

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
    agent.value,                                                   # path, required
    signer            = "Developer ID Installer: Example Corp (ABCDE12345)",  # required
    sha256            = "8974888d...",
    require_notarized = True,
    target            = "/",
    id                = "com.example.agent",
    version           = "1.2.0",
    label             = "install agent",                           # required
)
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `path` | string | required | The `.pkg` to install, positionally or as `path =`. An absolute path; a leading `~` expands to the home directory. |
| `signer` | string | required | The installer certificate's name, exactly as `pkgutil --check-signature` prints it: `Developer ID Installer: <developer> (<Team ID>)`. |
| `sha256` | string | none | The `.pkg` file's sha256 (`shasum -a 256`). When given, a package with any other content is refused before its signature is checked. That pins one exact build rather than any package the developer signed. |
| `require_notarized` | bool | `True` | Refuse packages Apple hasn't notarized. `None` counts as leaving it out, so it stays `True`; only `False` turns the check off. |
| `target` | string | `"/"` | The volume to install on (`installer -target`), as an absolute path. |
| `id`, `version` | string | none | Package id and version. When `pkgutil --pkg-info <id>` already reports this version on `target`, nothing is checked or installed and `changed = False`. Give both or neither. |

`install_mac_pkg` also takes the
[common arguments](../common-functionality.md#common-arguments): `label`
(required), `only_if`, `not_if`, `timeout` and
`what_if`. With `what_if = True` it checks the signature but doesn't install.
`timeout` doesn't stop `pkgutil` or `installer` once they've started.

## Finding the signer

Run `pkgutil --check-signature` on a copy of the package you trust, and copy
the name of the first certificate in the chain:

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
       ...
    2. Developer ID Certification Authority
       ...
    3. Apple Root CA
       ...
```

Here `signer` is `"Developer ID Installer: Tailscale Inc. (W5364U7YZB)"`.

## Returns

A [starcm_result](../common-functionality.md#starcm_result):

| Field | Value |
| --- | --- |
| `success` | `True` |
| `changed` | `True` when the package was installed (or, with `what_if`, would be); `False` when `id` / `version` was already installed |
| `message` | `installed <path>, signed by "<signer>"`, `would install ...` or `<id> <version> is already installed` |
| `value` | A `pkg_signature` struct with `signer`, `team_id`, `signer_sha256` (the certificate's fingerprint) and `notarized`. It's `None` when nothing was checked because the version was already installed. |

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
[install_remote_mac_pkg](install_remote_mac_pkg.md) does both steps in one
call, and doesn't fetch the package at all once that version is installed.

```python
# examples/install_mac_pkg/install.star
load("starcm", "download", "install_mac_pkg")

pkg = download(
    label  = "fetch agent",
    url    = "https://downloads.example.com/agent-1.2.0.pkg",
    sha256 = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
)

r = install_mac_pkg(
    pkg,
    # The installer certificate's name, from `pkgutil --check-signature`.
    signer  = "Developer ID Installer: Example Corp (ABCDE12345)",
    id      = "com.example.agent",
    version = "1.2.0",
    label   = "install agent",
)
print("%s (changed=%s)" % (r.message, r.changed))
```

### Check a package without installing it

```python
# examples/install_mac_pkg/preview.star
load("starcm", "install_mac_pkg", "vars")

r = install_mac_pkg(
    vars["pkg"],
    signer  = vars["signer"],
    what_if = True,
    label   = "check " + vars["pkg"],
)
print(r.message)
print("team %s, notarized=%s" % (r.value.team_id, r.value.notarized))
```

```bash
starcm --var pkg=Tailscale-1.98.8-macos.pkg --var "signer=Developer ID Installer: Tailscale Inc. (W5364U7YZB)" preview.star
```

```text
would install Tailscale-1.98.8-macos.pkg, signed by "Developer ID Installer: Tailscale Inc. (W5364U7YZB)"
team W5364U7YZB, notarized=True
```

Named with another developer, the same package is refused:

```bash
starcm --var pkg=Tailscale-1.98.8-macos.pkg --var "signer=Developer ID Installer: Example Corp (ABCDE12345)" preview.star
```

```text
Traceback (most recent call last):
  preview.star:3:20: in <toplevel>
Error in install_mac_pkg: install_mac_pkg(label="check Tailscale-1.98.8-macos.pkg"): Tailscale-1.98.8-macos.pkg: signed by "Developer ID Installer: Tailscale Inc. (W5364U7YZB)", not the expected "Developer ID Installer: Example Corp (ABCDE12345)"
```

### Unsigned packages are refused

These examples run in a scratch copy of
[examples/install_mac_pkg/](../../examples/install_mac_pkg). On macOS,
`pkgbuild` makes an unsigned package; elsewhere any file will do.

```scrut
$ cp -R "$TESTDIR/../../examples/install_mac_pkg/." "$TMPDIR" && cd "$TMPDIR" && if command -v pkgbuild >/dev/null; then pkgbuild --nopayload --identifier com.example.unsigned --version 1.0 unsigned.pkg >/dev/null 2>&1; else echo not a pkg > unsigned.pkg; fi
```

```python
# examples/install_mac_pkg/unsigned.star
load("starcm", "here", "install_mac_pkg")

install_mac_pkg(
    here("unsigned.pkg"),
    signer = "Developer ID Installer: Example Corp (ABCDE12345)",
    label  = "unsigned",
)
print("never reached")
```

```scrut
$ starcm unsigned.star 2>&1
Traceback (most recent call last):
  unsigned.star:3:16: in <toplevel>
Error in install_mac_pkg: install_mac_pkg\(label="unsigned"\): (.*[/\\]unsigned\.pkg: not signed with a certificate Apple issued for distribution: no signature|install_mac_pkg\(\) only works on macOS) (regex)
[1]
```

### A signer is required

```python
# examples/install_mac_pkg/empty_signer.star
load("starcm", "install_mac_pkg", "here")

install_mac_pkg(here("agent.pkg"), signer = "", label = "no signer")
```

```scrut
$ starcm empty_signer.star 2>&1
Traceback (most recent call last):
  empty_signer.star:3:16: in <toplevel>
Error in install_mac_pkg: install_mac_pkg(label="no signer"): signer must name the certificate, e.g. "Developer ID Installer: Example Corp (ABCDE12345)"
[1]
```

## Errors and edge cases

These are here mostly as tests: CI runs them to check error messages and
behaviour that the examples above don't show. They run in the copy of
[examples/install_mac_pkg/edge/](../../examples/install_mac_pkg/edge).

```scrut
$ cd "$TMPDIR/edge"
```

### id and version go together

```python
# examples/install_mac_pkg/edge/id_without_version.star
load("starcm", "here", "install_mac_pkg")

install_mac_pkg(
    here("agent.pkg"),
    signer = "Developer ID Installer: Example Corp (ABCDE12345)",
    id     = "com.example.agent",
    label  = "no version",
)
```

```scrut
$ starcm id_without_version.star 2>&1
Traceback (most recent call last):
  id_without_version.star:3:16: in <toplevel>
Error in install_mac_pkg: install_mac_pkg(label="no version"): id and version must be given together
[1]
```

## See also

- [download](download.md) to fetch the package first.
- [install_remote_mac_pkg](install_remote_mac_pkg.md) to fetch and install in one step.
- [verify_all](verify_all.md) to audit files signed with `starcm sign` instead.
