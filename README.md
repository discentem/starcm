![Static Badge](https://img.shields.io/badge/under%20development%2C%20not%20production%20ready-red?labelColor=yellow)

# starcm
"star-cm" ⭐

- A rudimentary configuration management language that utilizes Starlark instead of Ruby, json, or yaml.
- Why Starlark? Starlark provides variables, functions, loops, and lots more "for free" inside of the configuration files!

# Goal
Starcm is intended to become a viable alternative for tools like [macadmins/installapplications](https://github.com/macadmins/installapplications), [facebookincubator/go2chef](https://github.com/facebookincubator/go2chef), and [google/glazier](https://github.com/google/glazier).

# Installation

1. Download starcm from https://github.com/discentem/starcm/releases and install it somewhere in your path, such as `/usr/local/bin/starcm`.

> If you want to compile yourself, install [https://github.com/bazelbuild/bazelisk](Bazelisk) and run `make install`.

# What can Starcm do?

Starcm is very flexible and can accomplish lots of different tasks. Here are a just a few examples of what it can do.

#### Downloading a binary and verifying it's hash

```scrut
$ starcm examples/download/a_file.star
starcm_result(changed = True, error = "<nil>", label = "Downloading Ghostty 1.2.3", message = "downloaded file to Ghostty-1.2.3.dmg", return = None, success = True)
```

```python
# examples/download/a_file.star
download(
    label = "Download Ghostty 1.2.3",
    url = "https://release.files.ghostty.org/1.2.3/Ghostty.dmg",
    save_to = "Ghostty-1.2.3.dmg",
    sha256 = "f35ee91f116e28027ab9f8def45098c7575b44b407ff883a2dcd2985c483206b",
    live_progress = True
)
```

