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
starcm_result(changed = True, error = "<nil>", label = "Downloading Ghostty 1.2.3", message = "downloaded file to Ghostty-1.2.3.dmg", success = True, value = None)
```

```python
# examples/download/a_file.star
load("starcm", "download")

print(download(
    label = "Downloading Ghostty 1.2.3",
    url = "https://release.files.ghostty.org/1.2.3/Ghostty.dmg",
    save_to = "Ghostty-1.2.3.dmg",
    sha256 = "f35ee91f116e28027ab9f8def45098c7575b44b407ff883a2dcd2985c483206b",
    live_progress = False
))
```


#### Only running signed configuration

`starcm sign` writes a `<file>.sig` next to each file. Run with
`--trusted-keys DIR` (a directory of `*.pem` public keys; a key's id is its
file name before the first dot) and starcm refuses any `.star` file, including
ones pulled in by `load()` and `load_dynamic()`, without a valid signature
from one of those keys. `verify()` checks any other file the same way.

Every signature expires. `--expires` sets when, and the expiry is part of what's
signed, so an old release stops being accepted on its own instead of being
replayable forever. `--max-expiry 30d` also refuses signatures that expire more
than 30 days out, so a run only trusts recently signed files; with it, re-sign
on a schedule even when nothing changed.

```bash
openssl genpkey -algorithm ed25519 -out 2026-a.pem                # private key: keep it secret
openssl pkey -in 2026-a.pem -pubout -out trusted/2026-a.pub.pem   # public key: ship it
starcm sign --expires 30d --key 2026-a.pem main.star lib/winpe.star agent.msi
starcm --trusted-keys trusted --max-expiry 30d --var base=https://config.example.com main.star
```

```python
load("starcm", "download", "vars", "verify")

download(label = "fetch agent", url = vars["base"] + "/agent.msi", save_to = "agent.msi", sha256 = AGENT_SHA256)
check = verify("agent.msi", label = "check agent signature")   # reads agent.msi.sig
if not check.success:
    fail(check.error)
```

`--var name=value` (repeatable) passes values into a run as the `vars` dict.

#### Keeping downloads in a local store

Leave out `save_to` and `download()` saves into a local store instead, at
`<store>/sha256/<hash>/<name>` (named after the URL, or `file_name`), and
returns the stored path as `value`. A stored file is re-hashed every time it's
used, and starcm refuses a store that anyone other than the current user, root
or administrators could change.

The store defaults to `/Library/Application Support/starcm/store` (macOS),
`/var/lib/starcm/store` (Linux) or `%ProgramData%\starcm\store` (Windows) when
running as root or elevated, and to the user cache directory otherwise.
`--store-dir DIR` picks another, and `--store-only` refuses `save_to` so every
download goes into the store.

```python
load("starcm", "download", "exec")

agent = download(label = "fetch agent", url = "https://example.com/agent.pkg", sha256 = AGENT_SHA256)
exec(label = "install agent", cmd = "installer", args = ["-pkg", agent.value, "-target", "/"])
```

#### Checking configuration before it runs

`starcm inspect FILE...` prints what each file loads and calls, as JSON,
without running it. Calls are resolved through `load()` aliases, and
`load_dynamic` calls include their literal `path` (or `null` when it's built at
runtime), so a script can enforce rules such as "this file may only use
`download`" before signing or deploying:

```bash
starcm inspect bootstrap.star | jq -r '.[].calls[].function' | sort -u
```

The same checks can run in Starlark: `inspect(path)` returns that report as a
value, and `list_files(dir, pattern)`, `sha256(path)`, `json`,
`read_file(path)` and `sign(path, key = ..., expires = ...)` cover the rest of a signing
script. `sign()` is disabled when starcm runs with `--trusted-keys`, so a
deployed run can never sign anything.
