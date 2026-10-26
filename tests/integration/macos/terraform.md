# Terraform: download, extract and install a signed binary

Installs the real Terraform 1.17.0 CLI for Apple silicon into a scratch
directory with `download`, `extract` and `file(source = ..., signer = ...)`,
the flow an admin Mac's configuration uses. This runs only in the macOS
Install workflow (`.github/workflows/macos-install.yml`), because it
downloads 36 MB from releases.hashicorp.com and needs an Apple silicon Mac
and network access for the notarization check. It isn't part of the regular
scrut run.

```scrut
$ cp -R "$TESTDIR/fixtures/." "$TMPDIR" && cd "$TMPDIR" && export TMP="$TMPDIR" TEMP="$TMPDIR" && store="$TMPDIR/store" && bin="$TMPDIR/bin" && mkdir "$bin" && echo ready
ready
```

## A different signer is refused, and nothing is installed

```scrut
$ starcm --store-dir "$store" --var bin_dir="$bin" --var "signer=Developer ID Application: Example Corp (ABCDE12345)" terraform.star 2>&1 | tail -1; ls -A "$bin"
Error in file: file(label="install terraform"): *terraform: signed by "Developer ID Application: Hashicorp, Inc. (D38WU7D763)", not the expected "Developer ID Application: Example Corp (ABCDE12345)" (glob)
```

## The binary changed after extract() is refused

```scrut
$ starcm --store-dir "$store" --var bin_dir="$bin" --var tamper=yes terraform.star 2>&1 | grep '^Error'; ls -A "$bin"
Error in file: file(label="install terraform"): *terraform: codesign --verify: exit status 1 (glob)
```

## HashiCorp's signature installs it

```scrut
$ starcm --store-dir "$store" --var bin_dir="$bin" terraform.star
changed=True signer=Developer ID Application: Hashicorp, Inc. (D38WU7D763) notarized=True
```

```scrut
$ "$bin/terraform" version | head -1; stat -f %Lp "$bin/terraform"
Terraform v1.17.0
755
```

## Running it again changes nothing

```scrut
$ starcm --store-dir "$store" --var bin_dir="$bin" terraform.star
changed=False signer=Developer ID Application: Hashicorp, Inc. (D38WU7D763) notarized=True
```

## check_mac_signature reports the installed binary

```scrut
$ printf '%s\n' 'load("starcm", "check_mac_signature", "vars")' 'sig = check_mac_signature(vars["path"], signer = "Developer ID Application: Hashicorp, Inc. (D38WU7D763)")' 'print("signed by %r team=%s notarized=%s sha256=%s" % (sig.signer, sig.team_id, sig.notarized, sig.sha256))' > check.star && starcm --var path="$bin/terraform" check.star
signed by "Developer ID Application: Hashicorp, Inc. (D38WU7D763)" team=D38WU7D763 notarized=True sha256=6cb9932b1fbe2d461f98abd384e9488ea325f531137c7d83f01bde83e3dc52cc
```
