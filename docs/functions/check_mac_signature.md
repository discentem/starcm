# check_mac_signature

[← All functions](../README.md)

Use `check_mac_signature` to find out who signed a macOS installer package
(`.pkg`), command-line binary or `.app` bundle, and to stop the run unless
it's the developer your configuration names. It checks exactly what
[install_mac_pkg](install_mac_pkg.md) checks, and returns the same
`signer`, `team_id`, `signer_sha256` and `notarized`, plus the `sha256` of
the bytes it checked.

`check_mac_signature` is for reporting. To install something, don't check
it with `check_mac_signature` and then copy or run it in a separate step:
the file could be swapped in between. Use a function that checks the copy it
installs instead: [install_mac_pkg](install_mac_pkg.md) for packages and
[file](file.md#copy-a-signed-binary-into-place) with `source` and `signer`
for binaries.

The signature passes only when all of these hold:

- It was made with a certificate Apple issued for distribution: a Developer
  ID certificate, or Apple's own software signing. Apple only issues a
  Developer ID certificate to the team its name and Team ID belong to, so the
  name can be trusted. Unsigned and ad-hoc signed code is refused, and so are
  certificates from any other authority, even ones macOS trusts, because
  they could carry any name.
- The certificate chain ends at Apple's root certificate.
- The signing certificate, the first one in the chain, has exactly the name
  you gave, including the Team ID in parentheses.
- Apple has notarized it, unless you set `require_notarized = False`.
- Code (a binary or an app) hasn't been modified since it was signed.

## How it's checked

Packages are checked with `pkgutil --check-signature`, as
[install_mac_pkg](install_mac_pkg.md#finding-the-signer) does.

Binaries and apps are checked with `codesign`:

1. `codesign --verify --strict` (with `--deep` for an app) checks that the
   code hasn't changed since it was signed.
2. `codesign -dvvv` names the certificate chain (its `Authority=` lines)
   and the Team ID, and `codesign -d --extract-certificates` writes the
   certificates out so `signer_sha256` can be computed.
3. The names alone aren't trusted: anyone can make a certificate called
   "Apple Root CA" with their own authority, and a plain
   `codesign --verify` only checks code against a requirement its signer
   chose. So the code must also satisfy a
   [code requirement](https://developer.apple.com/library/archive/documentation/Security/Conceptual/CodeSigningGuide/RequirementLang/RequirementLang.html)
   that macOS checks against the certificates themselves. For Developer ID
   code it's the one Xcode gives Developer ID apps
   ([TN3127](https://developer.apple.com/documentation/technotes/tn3127-inside-code-signing-requirements)),
   with the signer's name added:

   ```text
   anchor apple generic
   and certificate 1[field.1.2.840.113635.100.6.2.6] exists      (issued by Developer ID Certification Authority)
   and certificate leaf[field.1.2.840.113635.100.6.1.13] exists  (a Developer ID Application certificate)
   and certificate leaf[subject.CN] = "<signer>"
   and certificate leaf[subject.OU] = "<Team ID>"
   ```

   For Apple's own code it's `anchor apple and certificate leaf[subject.CN] = "<signer>"`.
4. Notarization is checked with the `notarized` requirement,
   `codesign --verify --check-notarization -R=notarized`, which asks Apple
   online. Command-line binaries can't have a notarization ticket stapled to
   them, and `spctl` refuses to assess them ("does not seem to be an app"),
   so without a network connection Developer ID code counts as not
   notarized. Apple doesn't notarize its own code, so that needs
   `require_notarized = False`.

## Syntax

```python
load("starcm", "check_mac_signature")

check_mac_signature(
    "terraform",                                                  # path, required
    signer            = "Developer ID Application: Hashicorp, Inc. (D38WU7D763)",  # required
    require_notarized = True,
    kind              = "auto",                                   # or "pkg", "code"
)
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `path` | string | required | The package, binary or `.app` to check, positionally or as `path =`. An absolute path; a leading `~` expands to the home directory. |
| `signer` | string | required | The signing certificate's name: `Developer ID Application: <developer> (<Team ID>)` for code, `Developer ID Installer: ...` for packages. See [Finding the signer](#finding-the-signer). |
| `require_notarized` | bool | `True` | Refuse anything Apple hasn't notarized. |
| `kind` | string | `"auto"` | `"pkg"` to check an installer package with `pkgutil`, `"code"` to check a binary or app with `codesign`. `"auto"` picks `"pkg"` for paths ending in `.pkg` and `"code"` otherwise. |

`check_mac_signature` only reads, so it's a plain function: it takes no
`label` or other [common arguments](../common-functionality.md#common-arguments)
and returns the signature directly, not a `starcm_result`. It runs under
`what_if` too.

A file is copied into a private temporary directory and the copy is checked
and hashed, so `sha256` is the sha256 of exactly the bytes whose signature
was checked. An `.app` bundle is a directory, so it's checked where it is
and `sha256` is `""`. A file from the [local store](download.md#the-local-store)
is hashed again first, as [exec](exec.md#files-from-the-local-store) does.

`check_mac_signature` only works on macOS.

## Finding the signer

For code, copy the first `Authority=` line of `codesign -dvvv`:

```bash
codesign -dvvv terraform 2>&1 | grep -E '^(Authority|TeamIdentifier)='
```

```text
Authority=Developer ID Application: Hashicorp, Inc. (D38WU7D763)
Authority=Developer ID Certification Authority
Authority=Apple Root CA
TeamIdentifier=D38WU7D763
```

Here `signer` is `"Developer ID Application: Hashicorp, Inc. (D38WU7D763)"`.
For packages, see [install_mac_pkg](install_mac_pkg.md#finding-the-signer).

## Returns

A `mac_signature` struct with `signer`, `team_id`, `signer_sha256` (the
certificate's fingerprint), `notarized`, `kind` (`"pkg"` or `"code"`) and
`sha256` (of the file checked, `""` for an app).

A signature that fails any check, or a missing file, stops the run.

## Examples

Checking a Developer ID binary needs one downloaded from its developer, so
that example is shown but not run here;
[tests/integration/macos/terraform.md](../../tests/integration/macos/terraform.md)
runs it on a macOS runner. The others are run on Linux, macOS and Windows.
On macOS they check `/bin/ls`, and copies of it without a signature, with an
ad-hoc signature and with bytes added after signing. On Linux and Windows
the same calls fail because `check_mac_signature` only works on macOS.
Git Bash on Windows also rewrites `--var path=/bin/ls` to a Windows path
such as `C:/Program Files/Git/usr/bin/ls`, so the paths below allow a
prefix.

```scrut
$ cp -R "$TESTDIR/../../examples/check_mac_signature/." "$TMPDIR" && cd "$TMPDIR" && if command -v codesign >/dev/null; then cp /bin/ls unsigned && codesign --remove-signature unsigned && cp /bin/ls adhoc && codesign -f -s - adhoc 2>/dev/null && cp /bin/ls tampered && printf junk >> tampered; else for f in unsigned adhoc tampered; do echo not code > "$f"; done; fi
```

### Check who signed a binary

```bash
starcm --var path="$PWD/terraform" --var "signer=Developer ID Application: Hashicorp, Inc. (D38WU7D763)" check.star
```

```python
# examples/check_mac_signature/check.star
load("starcm", "check_mac_signature", "vars")

sig = check_mac_signature(
    vars["path"],
    signer            = vars["signer"],
    require_notarized = vars.get("require_notarized", "yes") == "yes",
)
print("signed by %r (notarized=%s, kind=%s)" % (sig.signer, sig.notarized, sig.kind))
```

```text
signed by "Developer ID Application: Hashicorp, Inc. (D38WU7D763)" (notarized=True, kind=code)
```

Named with another developer, the same binary is refused:

```text
Error in check_mac_signature: check_mac_signature: terraform: signed by "Developer ID Application: Hashicorp, Inc. (D38WU7D763)", not the expected "Developer ID Application: Example Corp (ABCDE12345)"
```

### Check Apple's own code

```python
# examples/check_mac_signature/report.star
load("starcm", "check_mac_signature")

# /bin/ls is Apple's own code. Apple doesn't notarize what ships with
# macOS, so require_notarized must be off.
sig = check_mac_signature(
    "/bin/ls",
    signer            = "macOS Software Signing",
    require_notarized = False,
)
print("signed by %r, notarized=%s, %d-digit sha256" % (sig.signer, sig.notarized, len(sig.sha256)))
```

```scrut
$ starcm report.star 2>&1 | tail -1
(signed by "macOS Software Signing", notarized=False, 64-digit sha256|Error in check_mac_signature: check_mac_signature: check_mac_signature\(\) only works on macOS) (regex)
```

With `require_notarized` left on, it's refused:

```scrut
$ starcm --var path=/bin/ls --var "signer=macOS Software Signing" check.star 2>&1 | tail -1
Error in check_mac_signature: check_mac_signature: (/bin/ls: not notarized by Apple; pass require_notarized = False to install it anyway|check_mac_signature\(\) only works on macOS) (regex)
```

So is another signer, even with `require_notarized = False`:

```scrut
$ starcm --var path=/bin/ls --var "signer=Developer ID Application: Example Corp (ABCDE12345)" --var require_notarized=no check.star 2>&1 | tail -1
Error in check_mac_signature: check_mac_signature: (/bin/ls: signed by "macOS Software Signing", not the expected "Developer ID Application: Example Corp \(ABCDE12345\)"|check_mac_signature\(\) only works on macOS) (regex)
```

### Unsigned, ad-hoc signed and modified code is refused

`go build` and `codesign -s -` make ad-hoc signatures: they say the code is
intact but not who made it.

```scrut
$ starcm --var path="$(pwd -W 2>/dev/null || pwd)/unsigned" --var "signer=macOS Software Signing" --var require_notarized=no check.star 2>&1 | tail -1
Error in check_mac_signature: check_mac_signature: (.*[/\\]unsigned: not signed with a certificate Apple issued for distribution: no signature|check_mac_signature\(\) only works on macOS) (regex)
```

```scrut
$ starcm --var path="$(pwd -W 2>/dev/null || pwd)/adhoc" --var "signer=macOS Software Signing" --var require_notarized=no check.star 2>&1 | tail -1
Error in check_mac_signature: check_mac_signature: (.*[/\\]adhoc: not signed with a certificate Apple issued for distribution: ad-hoc signature|check_mac_signature\(\) only works on macOS) (regex)
```

A binary changed after it was signed fails `codesign --verify`. The error
names the private copy that was checked:

```scrut
$ starcm --var path="$(pwd -W 2>/dev/null || pwd)/tampered" --var "signer=macOS Software Signing" --var require_notarized=no check.star 2>&1 | tail -2
(Error in check_mac_signature: check_mac_signature: .*[/\\]tampered: codesign --verify: exit status 1|  check\.star:3:\d+: in <toplevel>) (regex)
(.*/tampered: main executable failed strict validation|Error in check_mac_signature: check_mac_signature: check_mac_signature\(\) only works on macOS) (regex)
```

## Errors and edge cases

These are here mostly as tests: CI runs them to check error messages and
behaviour that the examples above don't show. They run in the copy of
[examples/check_mac_signature/edge/](../../examples/check_mac_signature/edge).

```scrut
$ cd "$TMPDIR/edge"
```

### A signer is required

```python
# examples/check_mac_signature/edge/empty_signer.star
load("starcm", "check_mac_signature", "here")

check_mac_signature(here("agent"), signer = "")
```

```scrut
$ starcm empty_signer.star 2>&1
Traceback (most recent call last):
  empty_signer.star:3:20: in <toplevel>
Error in check_mac_signature: check_mac_signature: signer must name the certificate, e.g. "Developer ID Application: Example Corp (ABCDE12345)"
[1]
```

### kind must be pkg, code or auto

```python
# examples/check_mac_signature/edge/bad_kind.star
load("starcm", "check_mac_signature", "here")

check_mac_signature(
    here("agent.dmg"),
    signer = "Developer ID Application: Example Corp (ABCDE12345)",
    kind   = "dmg",
)
```

```scrut
$ starcm bad_kind.star 2>&1
Traceback (most recent call last):
  bad_kind.star:3:20: in <toplevel>
Error in check_mac_signature: check_mac_signature: kind must be "auto", "pkg" or "code", not "dmg"
[1]
```

## See also

- [install_mac_pkg](install_mac_pkg.md) to check and install a package.
- [file](file.md#copy-a-signed-binary-into-place) to check and install a
  binary.
- [verify](verify.md) to check files signed with `starcm sign` instead.
