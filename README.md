![Static Badge](https://img.shields.io/badge/under%20development%2C%20not%20production%20ready-red?labelColor=yellow)

# starcm
"star-cm" ⭐

- A rudimentary configuration management language that utilizes Starlark instead of Ruby, json, or yaml.
- Why Starlark? Starlark provides variables, functions, loops, and lots more "for free" inside of the configuration files!

# Goal
Starcm is intended to become a viable alternative for tools like [macadmins/installapplications](https://github.com/macadmins/installapplications), [facebookincubator/go2chef](https://github.com/facebookincubator/go2chef), and [google/glazier](https://github.com/google/glazier).

# Installation

#### Option 1: Download a precompiled release

Download starcm from https://github.com/discentem/starcm/releases and install it somewhere in your path, such as `/usr/local/bin/starcm`.

#### Option 2: Compile 

Install [https://github.com/bazelbuild/bazelisk](Bazelisk) and do `make install`.

# What's possible with Starcm?

Starcm is very flexible and can accomplish lots of tasks. Here are a just a few examples of what it can do

## Download & install `.pkg` files on macOS

We can use Starcm to download and install packages for macOS. We can even store the configuration file (that tells Starcm what package we want to install) on a web server as well.

See [examples/install_go/bootstrap.star](examples/install_a_pkg_from_server/bootstrap.star) which shows an example of this.

You can run the example like so:

1. Start a web server that serves this repo.

    ```bash
    $ python3 -m http.server -d .
    ```
1. Run starcm with bootstrap.star, which will download an additional `.star` file from the webserver and execute it. Afterwards check that `/opt/example.json` now exists, which gets placed on disk by the package we installed.

    ```scrut
    $ starcm --root_file examples/install_a_pkg_from_server/bootstrap.star
    result(changed = True, diff = "", error = None, name = "download examplejson-1.0.pkg", output = "downloaded file to examplejson-1.0.pkg", success = True)

    {
        "name": "install examplejson-1.0.pkg",
        "changed": True,
        "success": True
    }
    ```