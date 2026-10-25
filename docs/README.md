# starcm documentation

Starcm configuration is written in [Starlark](https://github.com/google/starlark-go/blob/master/doc/spec.md).

But Starcm offers many additional functions that are not built into Starlark.


Each function below has its own page with its syntax, arguments, return
value and examples. 

Every example on these pages runs in CI with
[scrut](https://github.com/facebookincubator/scrut) on Linux, macOS and
Windows, so the output shown is the output you get.

## Functions

Load these from the `starcm` module, e.g. `load("starcm", "download", "exec")`.

All of these functions can be enhanced by:

- [Common functionality across starcm functions](common-functionality.md): the arguments every
  function takes (`label`, `only_if`, `not_if`, `timeout`,
  `what_if`), how paths work, and the `starcm_result` they return.
- [Loading additional files](loading.md): `load()`, the `starcm` and `stdlib` modules,
  and how paths resolve.
- [Command line flags](cli.md): `--var`, `--trusted-keys`, `--max-expiry`, `--release`, `--store-dir`, `--store-only`, `-v`, `--timestamps`,
  `starcm sign` and `starcm inspect`.
- [Design: paths and arguments](design/paths-and-arguments.md): the rule for
  relative paths, `here()`, and how arguments are parsed.


### Managing the machine

| Function | Description |
| --- | --- |
| [download](functions/download.md) | Download a file over HTTP(S), to a path or into a local store, and check its sha256. Does nothing if it's already there. |
| [s3_download](functions/s3_download.md) | The same, for an object in S3 or an S3-compatible store, with AWS credentials. |
| [exec](functions/exec.md) | Run a command, with extra environment variables if you like, capture its output and check its exit code. |
| [file](functions/file.md) | Create a file with given content or as a copy of another file (optionally only if it's signed by the macOS developer you name), or delete it. |
| [extract](functions/extract.md) | Unpack a `.zip`, `.tar.gz` or `.tgz` archive, refusing entries that would escape the destination. |
| [template](functions/template.md) | Render a Jinja2-style template to a file. |
| [install_mac_pkg](functions/install_mac_pkg.md) | Install a macOS `.pkg` only if it's signed by the certificate you pin. |
| [install_remote_mac_pkg](functions/install_remote_mac_pkg.md) | Fetch a `.pkg` over HTTP(S) or from S3 into the local store, pinned by sha256 or signature, and install it with the same checks. |

### Reading information

| Function | Description |
| --- | --- |
| [date](functions/date.md) | The current time, formatted and as fields, in any timezone. |
| [read_file](functions/read_file.md) | A file's content as a string. |
| [list_files](functions/list_files.md) | Files under a directory that match a pattern. |
| [glob](functions/glob.md) | Paths that match a pattern with wildcards in any part, such as `third_party/*/go.mod`. |
| [sha256](functions/sha256.md) | A file's sha256. |
| [inspect](functions/inspect.md) | What a `.star` file loads and calls, without running it. |
| [check_mac_signature](functions/check_mac_signature.md) | Who signed a macOS package, binary or app; stops the run unless it's the developer you name. |

### Signing

| Function | Description |
| --- | --- |
| [sign](functions/sign.md) | Sign a file with an ed25519 key (also `starcm sign`). |
| [verify](functions/verify.md) | Check a file's signature against `--trusted-keys`. |

### Output and control flow

| Function | Description |
| --- | --- |
| [write](functions/write.md) | Print a value, subject to `only_if` / `not_if`. |
| [load_dynamic](functions/load_dynamic.md) | Load a `.star` file whose path is computed at runtime. |
| [load_remote](functions/load_remote.md) | Fetch a `.star` file over HTTP(S) or from S3, pinned by sha256 or signature, and load it. |

### Values and helpers

These take no `label` and don't return a `starcm_result`.

| Name | Description |
| --- | --- |
| [json](functions/json.md) | `json.encode`, `json.decode` and `json.indent`. |
| [path](functions/path.md) | `path.exists` (is a file or directory there?) and `path.join` (build a path from parts). |
| [vars](functions/vars.md) | The `--var name=value` values, as a dict. |
| [dedent](functions/dedent.md) | Take the shared indentation off a triple-quoted string. |
| [temp_dir](functions/temp_dir.md) | The system's temporary directory, with optional path parts joined on. |
| [platform](functions/platform.md) | The machine's operating system and CPU architecture, as `os` and `arch`. |
| [env](functions/env.md) | An environment variable's value, or a default when it isn't set. |
| [home_dir](functions/home_dir.md) | The user's home directory, with optional path parts joined on. |
| [here](functions/here.md) | The directory of the `.star` file the call is written in, with optional path parts joined on. |
| [struct](functions/struct.md) | Build a record with named fields. Load it from `stdlib`: `load("stdlib", "struct")`. |

Starlark's built-ins, such as `print`, `fail`, `len`, `range` and `sorted`,
are always available.

## These pages are tests

Every page here is also a [scrut](https://github.com/facebookincubator/scrut)
test. CI runs every ```` ```scrut ```` code block and checks that the command
prints exactly the output shown, on Linux, macOS and Windows, so the examples
can't go out of date.

- **Examples** come first on each page and show typical use.
- **Errors and edge cases**, at the bottom of a page, are there mostly as
  tests: error messages, tampered signatures, arguments that must not leak
  between calls, and so on.

The `.star` files the pages run live in [examples/](../examples), one folder
per page (`examples/download/` for [download](functions/download.md), and so
on), so you can browse and copy them. The files only the "Errors and edge
cases" section uses are in that folder's `edge/` subfolder. Each page copies
its folder into a scratch directory before running anything, so the examples
never change the files in `examples/`. The test-only signing keys the
signature examples use are in [keys/](keys).

Where a page shows an example's source, the first line of the code block is
a comment naming the file, such as `# examples/download/basic.star`. The
Docs workflow runs [check_sources.py](check_sources.py) to make sure each such
block still matches its file.

To run the pages locally, build starcm, put it on your `PATH`, start the HTTP
server the download examples use, and run scrut from the repository root:

```bash
make install
python3 -m http.server 8000 --bind 127.0.0.1 &
scrut test --work-directory . docs/
python3 docs/check_sources.py
```

Two sets of tests need more than that, so they run in their own CI jobs:

- The quick start in the top-level README.md installs a real package with
  `sudo` (the macOS Install workflow).
- [tests/integration/s3/chain.md](../tests/integration/s3/chain.md) runs a chain of
  `.star` files from a real S3-compatible server, rustfs.
