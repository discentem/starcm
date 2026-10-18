# install_mac_pkg()

Installing needs a package signed by a real Developer ID, so the install
itself is covered by the Go unit tests in `functions/macpkg`, which use real
`pkgutil --check-signature` output. These tests check argument errors, which
are the same on every OS, and that an unsigned package is refused on macOS.
On other systems the same call fails because `install_mac_pkg` only works on
macOS.

```scrut
$ cp -R "$TESTDIR/fixtures/install_mac_pkg/." "$TMPDIR" && cd "$TMPDIR"
```

## signer_sha256 must be a sha256 fingerprint

```scrut
$ starcm bad_fingerprint.star 2>&1
Traceback (most recent call last):
  bad_fingerprint.star:4:16: in <toplevel>
Error in install_mac_pkg: install_mac_pkg(label="team id"): signer_sha256: "W5364U7YZB" is not a sha256 fingerprint (64 hex digits)
[1]
```

## id and version go together

```scrut
$ starcm id_without_version.star 2>&1
Traceback (most recent call last):
  id_without_version.star:3:16: in <toplevel>
Error in install_mac_pkg: install_mac_pkg(label="no version"): id and version must be given together
[1]
```

## Unsigned packages are refused

On macOS, `pkgbuild` makes an unsigned package; elsewhere any file will do.

```scrut
$ if command -v pkgbuild >/dev/null; then pkgbuild --nopayload --identifier com.example.unsigned --version 1.0 unsigned.pkg >/dev/null 2>&1; else echo not a pkg > unsigned.pkg; fi
```

```scrut
$ starcm unsigned.star 2>&1
Traceback (most recent call last):
  unsigned.star:3:16: in <toplevel>
Error in install_mac_pkg: install_mac_pkg\(label="unsigned"\): (unsigned\.pkg: signature is not trusted: no signature|install_mac_pkg\(\) only works on macOS) (regex)
[1]
```
