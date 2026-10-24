# Design: loading remote folders

[← Documentation index](../README.md)

Status: implemented. See [download](../functions/download.md#folders),
[s3_download](../functions/s3_download.md#folders),
[load_remote](../functions/load_remote.md#folders) and
[starcm sign --dir](../cli.md#starcm-sign---dir-dir). The
[implementation notes](#implementation-notes) at the end say where it
differs from, or adds to, the design below.

## Problem

`load_remote` fetches one `.star` file, pinned by sha256 or a signature, and
runs it from starcm's store. Nothing ships next to it there, so a remote
module can't use [here()](../functions/here.md) or a relative bundled-resource
path (`template`'s `template`, `load_dynamic`'s module, `exec`'s `cmd`): both
are errors in a store module (see
[paths and arguments](paths-and-arguments.md)). It has to fetch each
template, script or settings file itself with `download` or `s3_download`,
each pinned on its own.

So a configuration that works from a folder on disk breaks once it's
published, unless it's rewritten to fetch its files one by one. Zipping the
folder would fix that, but publishing should be "upload the folder", not
"build an archive".

## Decision

`download`, `s3_download` and `load_remote` can fetch a folder: a URL or S3
prefix ending in `/`, plus a signed manifest that lists every file in it with
its sha256. `download` and `s3_download` fetch the folder; `load_remote` fetches
it the same way and then runs a module from it.

### Publishing

```bash
starcm sign --dir config/agent/ --key release.pem --expires 30d
aws s3 sync config/agent/ s3://config/agent/
```

`starcm sign --dir DIR` walks DIR and writes two files into it:

- `starcm-manifest.json`: every file's path relative to DIR (slash
  separated) and its sha256.

  ```json
  {
    "version": 1,
    "files": {
      "main.star": "4aad939f…",
      "agent.conf": "fde98a8c…",
      "templates/motd.jinja": "5e5d7344…"
    }
  }
  ```

- `starcm-manifest.json.sig`: an ordinary starcm signature of the manifest,
  with the same key ids, expiry and release as any other signature.

Files and directories whose names start with `.`, and the manifest and its
signature, are left out. Individual files don't need their own `.sig`.

### Loading

```python
r = load_remote(
    "s3://config/agent/",
    module    = "main.star",
    signature = "s3://config/agent/starcm-manifest.json.sig",
    label     = "agent config",
)
```

- **Arguments.** `url` ends in `/` and names the folder; `module` is the
  module to run, a path in the manifest; `signature` or `sha256` pins the
  manifest, exactly as they pin a single file today. HTTP(S) and `s3://`
  work the same way: the manifest says what to fetch, so no directory
  listing is needed.
- **Steps.**
  1. Fetch the manifest and check it against `sha256` or, with
     `--trusted-keys`, its signature.
  2. Fetch each listed file, relative to `url`, and check it against the
     manifest's sha256. A missing file or a wrong hash stops the run.
     Files in the folder but not in the manifest are never fetched.
  3. Lay the files out as a folder in the store and run `module` there.
- **Inside the folder** `here()`, relative bundled-resource paths and
  relative `load()` statements resolve within it, so the folder runs the
  same from the store as it does from a checkout. `load()` and resource
  paths can't leave it: `..` past its root is an error.
- **Once per run**, as today: loading the same manifest again returns the
  same globals.

### Downloading a folder

```python
agent = download(
    label     = "fetch agent files",
    url       = "https://config.example.com/agent/",
    folder    = True,
    signature = "https://config.example.com/agent/starcm-manifest.json.sig",
)
# agent.value is the folder's absolute path in the store

s3_download(
    label     = "fetch agent files",
    bucket    = "config",
    key       = "agent/",
    signature = "agent/starcm-manifest.json.sig",
    save_to   = vars["managed_dir"] + "/agent",
)
```

- `download(folder = True)`, or an S3 `key` ending in `/`, names a folder.
  `signature` or `sha256` pins its manifest; there's no per-file pin.
- Without `save_to`, the folder is assembled in the store (below) and
  `value` is its absolute path.
- With `save_to`, an absolute directory, each listed file is written there
  the way `download` writes one file today: to a temporary file, checked,
  then renamed into place, so no file ever holds unverified data. Files
  already there with the right hash aren't fetched again. Files in
  `save_to` that the manifest doesn't list are left alone. `changed` is
  `True` when any file was written. `value` is `save_to`.
- `file_name` doesn't apply to folders.
- Both use the same `download.Fetcher` call, `FetchFolder`, which
  `load_remote` uses too.

### Store layout

Each file is stored once under its own sha256, as downloads are today. The
folder is assembled under the manifest's sha256:

```text
<store>/sha256/<file hash>/<name>                      each file, once
<store>/folders/<manifest hash>/starcm-manifest.json
<store>/folders/<manifest hash>/starcm-manifest.json.sig  when pinned by signature
<store>/folders/<manifest hash>/main.star              hard link, or copy
<store>/folders/<manifest hash>/templates/motd.jinja
```

A new release only downloads the files whose hashes changed. Before running,
starcm re-hashes the assembled folder against the manifest, as it re-checks
stored files today, so a file changed after it was stored stops the run.

### Trust

One signature, on the manifest, covers every file: a file's bytes must match
the hash the signed manifest lists. The loader's `--trusted-keys` check
accepts a `.star` file inside an assembled folder when the folder's manifest
is verified, instead of looking for a `.sig` next to each module.

## What has to exist first

Already there:

- `download.Fetcher` (`FetchURL`, `FetchS3`), pinning and signatures
  ([paths and arguments](paths-and-arguments.md), part 2).
- The local store with `--store-dir` and its re-check of stored files.
- `starcm sign` and the `--trusted-keys` keyring.
- The store check behind the `here()` and relative-resource errors in remote
  modules.

New:

1. **Manifest format and `starcm sign --dir`**, in `libraries/signing`
   (write, parse, check against a folder).
2. **Store folders**: `Store.PutFolder(manifestHash, files)` and a re-check,
   in `libraries/local_store`.
3. **`Fetcher.FetchFolder`**: fetch the manifest and each listed file
   relative to a folder URL or S3 prefix, into the store or `save_to`.
   `download` and `s3_download` accept folder URLs through it.
4. **`load_remote` folders**: the `module` argument, on top of `FetchFolder`.
5. **Loader**: run `module` from the folder; trust `.star` files in a
   verified folder under `--trusted-keys`; keep `load()` and resource paths
   inside the folder; stop treating a verified folder as "nothing ships next
   to it" in the store check.
6. **Docs and tests**: folder sections on the `download`, `s3_download` and
   `load_remote` pages, HTTP examples, and a folder case in the S3
   integration test.

## Not doing

- **A Starlark manifest** (a `.star`-like file holding a dict or struct):
  the manifest is data that `starcm sign --dir` writes and starcm reads
  back; nobody edits hashes by hand. Reading a `.star` file means running
  it, before any file it pins is checked, and lets a manifest compute its
  list. With a `.star` name, `--trusted-keys`, `inspect` and
  `list_files("**/*.star")` would treat it as a module; with another name
  it's Starlark syntax without Starlark. JSON is parsed strictly and other
  tools (CI, `jq`) can read it.

- **Directory listings** (HTML indexes or S3 `ListObjects`) instead of a
  manifest: they can't be pinned, and anyone who can write to the bucket
  could add a file.
- **Archives** (`.tar.gz`): a manifest gives the same single pin without
  building anything, and changed files download on their own.
- **Partial uploads**: a machine that fetches mid-upload sees a hash
  mismatch and fails cleanly; it never runs a mix of releases. Upload the
  manifest and its signature last.

## Implementation notes

What the implementation settled that the design above left open:

- **Manifest rules.** `version` must be 1 and unknown fields are refused.
  Paths are relative and slash separated, with no empty, `.` or `..` parts,
  no `\`, `:` or NUL, and not the manifest or its signature. Two paths that
  differ only in case, or a path that's also another's directory, are
  refused, since they can't be laid out on every platform; `starcm sign
  --dir` refuses such a folder rather than publishing it. Anything in the
  folder that isn't a regular file or directory, such as a symlink, is an
  error when signing.
- **Store files.** Each file is stored as `<store>/sha256/<hash>/<name>`,
  `<name>` being the last part of its path, so identical files in two
  folders, or two releases, are stored once.
- **Re-check.** `GetFolder` re-hashes every file and the manifest, and also
  refuses a folder holding files its manifest doesn't list, or missing one.
  The loader checks each module's bytes against the manifest again right
  before running it, and `exec` re-checks a script from a folder. A folder
  is assembled in a temporary directory and renamed into place, so
  `<store>/folders/<hash>/` is never partial.
- **`save_to`.** The manifest, and its signature when pinned by one, are
  written into `save_to` after the files, as `download` keeps a file's
  `.sig` next to it. With the manifest already there, it's used to check
  the files instead of being fetched again.
- **`--trusted-keys` and `sha256`.** A folder pinned by `sha256` has no
  signature in the store, so its modules are refused under
  `--trusted-keys`, as a single module pinned by `sha256` is.
- **`download(folder = True)`.** A folder is named twice, by
  `folder = True` and by the URL's trailing `/`, and either without the
  other is an error, so a slip can't fetch the wrong thing. (`s3_download`
  and `load_remote` still go by the trailing `/` alone.)
- **`module`** (named `entry` in the first draft of this note; renamed to
  match `load_dynamic`'s `module`) must be a `.star` path the manifest
  lists, and is an error for a URL that isn't a folder.
- **Once per run.** The loader caches modules by the path they resolve to
  rather than the string passed to `load()`, so `load("lib/x.star")` in a
  fetched folder and in the local checkout are two modules.
- **Escaping the folder.** `..` past the folder's root is refused
  lexically, for `load()`, `here()` and relative resource paths. Absolute
  paths, and `//` workspace paths in `load()`, mean what they mean outside
  a folder.
