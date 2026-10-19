![Static Badge](https://img.shields.io/badge/under%20development%2C%20not%20production%20ready-red?labelColor=yellow)

# starcm
"star-cm" ⭐

- A rudimentary configuration management language that utilizes Starlark instead of Ruby, json, or yaml.
- Why Starlark? Starlark provides variables, functions, loops, and lots more "for free" inside of the configuration files!

# Goal
Starcm is intended to become a viable alternative for tools like [macadmins/installapplications](https://github.com/macadmins/installapplications), [facebookincubator/go2chef](https://github.com/facebookincubator/go2chef), and [google/glazier](https://github.com/google/glazier).

# Installation

1. Download starcm from https://github.com/discentem/starcm/releases and install it somewhere in your path, such as `/usr/local/bin/starcm`.

> If you want to compile yourself, install [Bazelisk](https://github.com/bazelbuild/bazelisk) and run `make install`.

# Quick start

starcm runs `.star` files. Here is an example of a `.star` file that installs
a `.pkg` file on macOS: [dockutil](https://github.com/kcrawford/dockutil). It
downloads the package, checks it against a pinned sha256, and installs it only
if it's signed by dockutil's developer and notarized by Apple.

Save the code below as `install_dockutil.star`. (The same file is in this
repository at
[examples/quickstart/install_dockutil.star](examples/quickstart/install_dockutil.star).)
The package is kept in starcm's
[local store](docs/functions/download.md#the-local-store), so nothing is
left in the directory you run it from.

```python
# examples/quickstart/install_dockutil.star
load("starcm", "download", "install_mac_pkg")

# dockutil 3.1.3. download() checks the sha256 and keeps the package in
# starcm's local store.
pkg = download(
    label  = "fetch dockutil",
    url    = "https://github.com/kcrawford/dockutil/releases/download/3.1.3/dockutil-3.1.3.pkg",
    sha256 = "f60db8273fe80d7c4824588bedb538a5387675c3917edb0010dab9f52d3f2582",
)

# Install it only if dockutil's developer signed it and Apple notarized it.
r = install_mac_pkg(
    pkg.value,
    signer  = "Developer ID Installer: Kyle Crawford (Z5J8CJBUWC)",
    id      = "dockutil.cli.tool",
    version = "3.1.3",
    label   = "install dockutil",
)
if r.changed:
    print("installed dockutil, signed by %s" % r.value.signer)
else:
    print(r.message)
```

Installing a package needs root, so run it with `sudo`:

```scrut
$ sudo starcm install_dockutil.star
installed dockutil, signed by Developer ID Installer: Kyle Crawford (Z5J8CJBUWC)
```

Run it again and nothing changes:

```scrut
$ sudo starcm install_dockutil.star
dockutil.cli.tool 3.1.3 is already installed
```

# Documentation

**[docs/](docs/README.md)** lists every function starcm provides, each with
its arguments, return value and tested examples:

- Manage the machine with [download](docs/functions/download.md) (to a
  path or into a local, hash-checked store),
  [s3_download](docs/functions/s3_download.md),
  [exec](docs/functions/exec.md), [file](docs/functions/file.md),
  [template](docs/functions/template.md) and
  [install_mac_pkg](docs/functions/install_mac_pkg.md).
- Read information with [date](docs/functions/date.md),
  [read_file](docs/functions/read_file.md),
  [list_files](docs/functions/list_files.md),
  [sha256](docs/functions/sha256.md) and
  [inspect](docs/functions/inspect.md).
- Run only signed configuration with [sign](docs/functions/sign.md),
  [verify](docs/functions/verify.md) and
  [--trusted-keys](docs/cli.md#--trusted-keys-dir).
- Pass values in with [--var](docs/functions/vars.md), and split
  configuration across files with [load()](docs/loading.md) and
  [load_dynamic](docs/functions/load_dynamic.md).
