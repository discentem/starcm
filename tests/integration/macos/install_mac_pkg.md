# install_mac_pkg(): real install

Installs a real signed, notarized package as root, then removes it. This
runs only in the macOS Install workflow (`.github/workflows/macos-install.yml`),
which has passwordless `sudo`. It isn't part of the regular scrut run.

The package is Installomator 10.9 (145 KB, one file, no install scripts).

```scrut
$ cp -R "$TESTDIR/fixtures/." "$TMPDIR" && cd "$TMPDIR" && starcm="$(command -v starcm)" && test ! -e /usr/local/Installomator && echo "not installed yet"
not installed yet
```

## A different signer is refused

```scrut
$ sudo "$starcm" --var "signer=Developer ID Installer: Example Corp (ABCDE12345)" installomator.star 2>&1 | tail -1; test ! -e /usr/local/Installomator && echo "nothing installed"
Error in install_mac_pkg: install_mac_pkg(label="install Installomator"): *Installomator-10.9.pkg: signed by "Developer ID Installer: Armin Briegel (JME5BW3F3R)", not the expected "Developer ID Installer: Example Corp (ABCDE12345)" (glob)
nothing installed
```

## A different sha256 is refused

```scrut
$ sudo "$starcm" --var pkg_sha256=0000000000000000000000000000000000000000000000000000000000000000 installomator.star 2>&1 | tail -1; test ! -e /usr/local/Installomator && echo "nothing installed"
Error in install_mac_pkg: install_mac_pkg(label="install Installomator"): *Installomator-10.9.pkg has sha256 8974888d6b27071cd6418d2d5afc53e39f7dff682641e7faed64d3d379679fad, not the expected 0000000000000000000000000000000000000000000000000000000000000000 (glob)
nothing installed
```

## The expected signer installs it

```scrut
$ sudo "$starcm" installomator.star
installed *Installomator-10.9.pkg, signed by "Developer ID Installer: Armin Briegel (JME5BW3F3R)" | changed=True (glob)
team=JME5BW3F3R notarized=True
```

```scrut
$ test -x /usr/local/Installomator/Installomator.sh && pkgutil --pkg-info com.scriptingosx.Installomator | head -2
package-id: com.scriptingosx.Installomator
version: 10.9
```

## Installing again does nothing

```scrut
$ sudo "$starcm" installomator.star
com.scriptingosx.Installomator 10.9 is already installed | changed=False
```

## Clean up

```scrut
$ sudo rm -rf /usr/local/Installomator && sudo pkgutil --forget com.scriptingosx.Installomator
Forgot package 'com.scriptingosx.Installomator' on '/'.
```
