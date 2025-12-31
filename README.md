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

Starcm is very flexible and can accomplish lots of tasks. Here are a just a few examples of what it can do.

## Downloading arbitrary files and verifying their hashes

