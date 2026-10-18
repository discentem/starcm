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

A configuration is a `.star` file that calls starcm's functions. This one
makes sure a file has the right content, and shows it only when it changed.

Save the code below as `motd.star`, in any directory, and run the commands
after it. (The same file is in this repository at
[docs/fixtures/quickstart/motd.star](docs/fixtures/quickstart/motd.star).)
`motd.star` calls [temp_dir()](docs/functions/temp_dir.md), which returns
your system's temporary directory, so `motd.txt` goes into a
`starcm-quickstart` folder there instead of the directory you run it from.

`motd.star`:

<!-- fixture: docs/fixtures/quickstart/motd.star -->
```python
load("starcm", "exec", "file", "temp_dir")

# A starcm-quickstart folder in the system's temporary directory.
path = temp_dir("starcm-quickstart", "motd.txt")

motd = file(
    label       = "message of the day",
    path        = path,
    content     = "Managed by starcm. Changes will be overwritten.\n",
    create_dirs = True,
)
print("motd: changed=%s" % motd.changed)

exec(
    label   = "show motd",
    cmd     = "cat",
    args    = [path],
    only_if = motd.changed,
    live_output = True,
)
```

Run it:

```scrut
$ starcm motd.star
motd: changed=True
Managed by starcm. Changes will be overwritten.
```

Run it again and nothing changes:

```scrut
$ starcm motd.star
motd: changed=False
INFO: [show motd]: skipping shell(label="show motd") because only_if was false
```

# Documentation

**[docs/](docs/README.md)** lists every function starcm provides, each with
its arguments, return value and tested examples:

- Manage the machine with [download](docs/functions/download.md) (to a
  path or into a local, hash-checked store),
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
