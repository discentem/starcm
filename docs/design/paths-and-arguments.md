# Design: paths and arguments

[← Documentation index](../README.md)

Status: accepted. Implemented.

## Problem

- **Arguments are parsed twice.** `base.Module` validates every call with
  `starlark.UnpackArgs`, then each function re-reads the raw `args`/`kwargs`
  itself. The two layers drift: most functions only look at keyword
  arguments, so positional calls pass validation and then fail or are
  ignored (`file`, `template`, `exec`, `extract` with a positional `dest`,
  and `date`, which silently ignores a positional `format`).
- **Relative paths mean different things in different functions.** Only
  `template` resolves them against the calling file's directory (via
  `working_directory`); everything else uses the directory starcm was
  started in, which under MDM or launchd is usually `/`. Only some
  functions expand `~`.

## Decision

### One rule for paths

**A relative path means one thing only: a read-only resource that ships next
to the `.star` file.** Everything else must be absolute.

| Kind | Arguments | Relative path |
| --- | --- | --- |
| Bundled resources | `template(template)`, `load_dynamic(module)` | Next to the `.star` file the call is written in |
| Outputs | `file(path)`, `template(destination)`, `download(save_to)`, `extract(dest)`, files `sign` writes | Error: must be absolute |
| Other inputs | `read_file`, `sha256`, `inspect`, `path.list`, `path.glob`, `path.exists`, `extract(archive)`, `install_mac_pkg`, `verify`, `file(source)` | Error: must be absolute |

- When you leave a file behind you know where it should go, so outputs
  name it. There is no working directory to resolve against: the
  `working_directory` common argument is removed, and nothing resolves
  against the directory starcm was started in.
- Inputs that could be either bundled or produced by an earlier step are
  absolute, so the same argument never means two places. Pass the result
  of the step that produced the file (its value is an absolute path), or
  build the path with a helper.
- `~` is expanded in every path argument.
- Every path a function returns is absolute.
- The [store](../functions/download.md#the-local-store) is unaffected: it is
  a fixed, verified location chosen by `--store-dir`.

### Helpers that build absolute paths

- `here(*parts)`: `parts` joined to the directory of the `.star` file the
  `here()` call is written in. Call it in the file that sits next to the
  resource:

  ```python
  # site/web.star (motd.jinja is next to this file)
  load("//lib/render.star", "render")
  render(here("motd.jinja"), "/etc/motd")   # absolute before render() sees it
  ```

  `here()` is for read-only files that ship with the code, never for
  outputs. In a module loaded with `load_remote`, whose directory is
  starcm's store, it is an error, and so is any relative bundled-resource
  path.
- `home_dir(*parts)`, `temp_dir()`: unchanged.

### exec runs somewhere explicit

What directory a command runs in isn't about files left behind, so `exec`
gets its own optional absolute `cwd` argument. Without it the command
inherits starcm's directory, as today.

### Arguments are parsed once

`base` checks each call's arguments once and hands the function a
`base.FunctionInstance`: the checked arguments of that one use of the
function. The function reads them with `base.Get[T](inst, name)`,
`inst.Given(name)`, and the path accessors that apply the rule above,
`inst.AbsPath(name)` and `inst.ResourcePath(name)`. Positional and keyword
arguments then behave identically for every function, errors read the same
everywhere, and a rule like "expand `~`" lives in one place. Each function
implements `Execute(inst *base.FunctionInstance)`.
`functions/argcompat` checks that every function gives the same result
called positionally and by keyword.

### Shared logic is Go

When two Starlark functions do the same work, the work lives in a typed Go
API and each function is a thin adapter that reads its arguments and calls
it. Functions never call each other by building Starlark arguments.
`download`, `s3_download` and `load_remote` share `download.Fetcher`
(`FetchURL`, `FetchS3`, with a `Request` and a `Fetched` result);
`load_remote` calls it directly rather than running `download()`.

### Results passed as arguments

Once arguments are parsed in one place, a `starcm_result` passed directly as
an argument is replaced by its `value` (including inside a list or dict),
so `extract(archive = zip)` works without `.value`. A result that didn't
succeed stops the run instead. In `only_if` and `not_if` a result stands
for its `success` instead of its value, and one that didn't succeed is
`False` rather than an error, so `not_if = check` runs the step when `check`
failed.

### Steps and queries

A function is a **step** if it changes the machine, or runs something whose
effect starcm can't know: `file`, `template`, `download`, `s3_download`,
`extract`, `exec`, `install_mac_pkg`, `install_remote_mac_pkg`, `sign`,
`write`, `load_dynamic`, `load_remote`. Steps take a `label` and the common
arguments and return a `starcm_result`, since that's where naming a step in
the log, gating it with `only_if`/`not_if`, a `timeout` and `what_if` mean
something.

Every function that only reads is a **query**: `path.exists`, `path.join`,
`path.glob`, `path.list`, `read_file`, `sha256`, `inspect`, `date`,
`verify`, `check_mac_signature`, and the helpers (`here`, `home_dir`,
`temp_dir`, `env`, `platform`, `dedent`). A query takes no common arguments
and returns its answer directly, so it works in `if`, `for` and
comprehensions without `.value`. Any error stops the run. It runs under
`what_if` like any other read, so a dry run sees the real state.

The line is whether the call changes something, not whether it touches the
filesystem: `path.exists` and `path.glob` read the disk but are queries.

Queries use `base.Query`, which checks arguments the same way as steps
(through `FunctionInstance`, so the path rule, `~` expansion, positional
arguments and result unwrapping are shared) and adds no common arguments.
Helpers that take `*parts` (`path.join`, `here`, `home_dir`, `temp_dir`)
stay hand-written builtins, since `UnpackArgs` has no varargs.

## Consequences

- Breaking: relative outputs and inputs (other than bundled resources) stop
  working, and `working_directory` is removed. The error names the fix
  (a full path, or the result of the step that made the file).
- Positional arguments fill parameters in the order each function declares
  them, so `write("a", "b")` now means `end = "b"` instead of being an error.
- Rollout: (1) this note, `here()`, the path helpers and
  `base.FunctionInstance`; (2) every function moved to
  `base.FunctionInstance` and the path rule, the old interface and
  `working_directory` removed; (3) results unwrapped when passed as
  arguments; (4) read-only functions became queries (`list_files` became
  `path.list`), and a result in `only_if`/`not_if` stands for its
  `success`.
- Breaking: queries no longer take `label` or return a `starcm_result`, so
  `read_file(p, label = "x").value` becomes `read_file(p)`, and `verify()`
  returns a `verification` struct whose `verified` replaces `success`.
