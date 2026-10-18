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
  `working_directory`, `what_if`) and the `starcm_result` they return.
- [Loading additional files](loading.md): `load()`, the `starcm` and `stdlib` modules,
  and how paths resolve.
- [Command line flags](cli.md): `--var`, `--trusted-keys`, `--max-expiry`, `--store-dir`, `--store-only`, `-v`, `--timestamps`,
  `starcm sign` and `starcm inspect`.


### Managing the machine

| Function | Description |
| --- | --- |
| [download](functions/download.md) | Download a file, to a path or into a local store, and check its sha256. Does nothing if it's already there. |
| [exec](functions/exec.md) | Run a command, capture its output and check its exit code. |
| [file](functions/file.md) | Create a file with given content, or delete it. |
| [template](functions/template.md) | Render a Jinja2-style template to a file. |
| [install_mac_pkg](functions/install_mac_pkg.md) | Install a macOS `.pkg` only if it's signed by the certificate you pin. |

### Reading information

| Function | Description |
| --- | --- |
| [date](functions/date.md) | The current time, formatted and as fields, in any timezone. |
| [read_file](functions/read_file.md) | A file's content as a string. |
| [list_files](functions/list_files.md) | Files under a directory that match a pattern. |
| [sha256](functions/sha256.md) | A file's sha256. |
| [inspect](functions/inspect.md) | What a `.star` file loads and calls, without running it. |

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

### Values and helpers

These take no `label` and don't return a `starcm_result`.

| Name | Description |
| --- | --- |
| [json](functions/json.md) | `json.encode`, `json.decode` and `json.indent`. |
| [vars](functions/vars.md) | The `--var name=value` values, as a dict. |
| [temp_dir](functions/temp_dir.md) | The system's temporary directory, with optional path parts joined on. |
| [struct](functions/struct.md) | Build a record with named fields. Load it from `stdlib`: `load("stdlib", "struct")`. |

Starlark's built-ins, such as `print`, `fail`, `len`, `range` and `sorted`,
are always available.

## Docs and tests

The pages here show typical use. [tests/](../tests) holds the edge cases:
error paths, tampered signatures, arguments leaking between calls and so on.
Both run in the same scrut CI job. To run them locally, build starcm, put it
on your `PATH`, start the HTTP server the download examples use, and run
scrut from the repository root:

```bash
make install
python3 -m http.server 8000 --bind 127.0.0.1 &
scrut test --work-directory . docs/ tests/
```

The quick start in the top-level README.md installs a real package with
`sudo`, so it isn't part of that run. The macOS Install workflow runs it on a
CI runner and removes the package afterwards.

Fixtures for the examples live in [fixtures/](fixtures). Each page copies
its fixtures into a scratch directory, so running the examples leaves
nothing behind.

An example's source is shown in a plain code block after a
`<!-- fixture: path -->` comment naming the file it runs. The Docs workflow
runs [check_sources.py](check_sources.py) to make sure each such block still
matches its file:

```bash
python3 docs/check_sources.py
```
