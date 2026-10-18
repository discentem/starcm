# download

[← All functions](../README.md)

Use `download` to fetch a file over HTTP(S) and check it against a pinned
sha256. The file goes either to a path you choose with `save_to`, or, without
`save_to`, into starcm's [local store](#the-local-store), and `value` is the
path it was stored at. If the file is already there with the right hash,
nothing is downloaded, so running the same configuration again is cheap and
reports `changed = False`.

## Syntax

```python
load("starcm", "download")

download(
    label         = "fetch agent",       # required
    url           = "https://...",       # required
    sha256        = "ffcf2b6c...",       # required
    save_to       = "agent.pkg",         # omit to use the local store
    file_name     = "agent.pkg",         # name in the store; only without save_to
    live_progress = False,
)
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `url` | string | required | URL to fetch. Any status other than `200 OK` fails the run. |
| `sha256` | string | required | Expected lowercase hex sha256 of the file. Use [sha256()](sha256.md) or `shasum -a 256` to get it. |
| `save_to` | string | none | Where to save the file, relative to the directory starcm runs in. Missing parent directories are created. Leave it out to download into the [local store](#the-local-store). Refused under [--store-only](../cli.md#--store-dir-dir-and---store-only). |
| `file_name` | string | the last part of the URL's path | The file's name in the local store. Only used without `save_to`. |
| `live_progress` | bool | `False` | Print a `Downloading <url>... N%` progress line while downloading. |

`download` also takes the [common arguments](../common-functionality.md#common-arguments):
`label` (required), `only_if`, `not_if`, `timeout`, `working_directory` and
`what_if`.

## What it does

With `save_to`:

1. If `save_to` exists and its sha256 matches, return `changed = False`.
2. Download `url` to a temporary file next to `save_to`, hashing as it goes.
3. If the hash doesn't match `sha256`, delete the temporary file and fail the
   run. Whatever was at `save_to` is left as it was.
4. Otherwise rename the temporary file to `save_to`, replacing any file
   there.

Without `save_to`, the same happens in the local store, except that a stored
file whose hash no longer matches stops the run instead of being replaced.

## The local store

The store keeps each download at `<store>/sha256/<sha256>/<file_name>`, so a
file is only ever stored under the hash it was checked against. Stored files
are read-only.

- **Where:** [--store-dir](../cli.md#--store-dir-dir-and---store-only) sets the
  store's directory. By default it's `/Library/Application
  Support/starcm/store` (macOS), `/var/lib/starcm/store` (Linux) or
  `%ProgramData%\starcm\store` (Windows) when starcm runs as root or
  elevated, and `starcm/store` in the user's cache directory otherwise. It's
  created the first time a run uses it.
- **Trusted:** starcm refuses a store, or any directory above it, that anyone
  other than the current user, root or administrators could change.
- **Re-checked:** a stored file is hashed again each time `download` finds it
  in the store, and right before [exec](exec.md) runs it, or runs with its
  path as one of `args`. A file changed after it was stored stops the run.
- **Only the store:** with
  [--store-only](../cli.md#--store-dir-dir-and---store-only), `save_to` is
  refused, so every download goes into the store.

## Returns

A [starcm_result](../common-functionality.md#starcm_result):

| Field | Value |
| --- | --- |
| `success` | `True` |
| `changed` | `True` when the file was downloaded, `False` when it was already present |
| `message` | `downloaded file to <path>`, `"<save_to>" already present and sha256 verified` or `"<path>" already in the store and sha256 verified` |
| `value` | Without `save_to`, the stored file's absolute path. With `save_to`, `None`. |

Network errors, HTTP errors and hash mismatches stop the run with an error
instead of returning a result.

## Examples

These examples download from the HTTP server that serves the repository root
on port 8000 (CI starts one with
`python3 -m http.server 8000 --bind 127.0.0.1`). Each runs in a scratch copy
of [docs/fixtures/download/](../fixtures/download).

```scrut
$ cp -R "$TESTDIR/../fixtures/download/." "$TMPDIR" && cd "$TMPDIR"
```

### Download a file and verify its hash

```scrut
$ cat basic.star
load("starcm", "download")

r = download(
    label   = "fetch agent",
    url     = "http://localhost:8000/docs/fixtures/download/files/agent-1.2.0.txt",
    save_to = "agent-1.2.0.txt",
    sha256  = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
)
print(r)
```

```scrut
$ starcm basic.star
starcm_result(changed = True, error = "<nil>", label = "fetch agent", message = "downloaded file to agent-1.2.0.txt", success = True, value = None)
```

Run it again and the existing file is verified instead of downloaded:

```scrut
$ starcm basic.star
starcm_result(changed = False, error = "<nil>", label = "fetch agent", message = "\"agent-1.2.0.txt\" already present and sha256 verified", success = True, value = None)
```

### Save into a directory that doesn't exist yet

```scrut
$ cat nested.star
load("starcm", "download")

r = download(
    label   = "fetch agent into a cache directory",
    url     = "http://localhost:8000/docs/fixtures/download/files/agent-1.2.0.txt",
    save_to = "cache/agent/1.2.0/agent.txt",
    sha256  = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
)
print(r.message)
```

```scrut
$ starcm nested.star && cat cache/agent/1.2.0/agent.txt
downloaded file to cache/agent/1.2.0/agent.txt
agent v1.2.0
```

### Take the server from `--var`

Keep the version and hash in the configuration and pass the environment in
with [--var](vars.md):

```scrut
$ cat from_var.star
load("starcm", "download", "vars")

VERSION = "1.2.0"
SHA256  = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08"

r = download(
    label   = "fetch agent %s" % VERSION,
    url     = "%s/agent-%s.txt" % (vars["base_url"], VERSION),
    save_to = "agent-%s.txt" % VERSION,
    sha256  = SHA256,
)
print("%s: success=%s changed=%s" % (r.label, r.success, r.changed))
```

```scrut
$ rm agent-1.2.0.txt && starcm --var base_url=http://localhost:8000/docs/fixtures/download/files from_var.star
fetch agent 1.2.0: success=True changed=True
```

### Pin a hash computed with `sha256()`

```scrut
$ cat pin_with_sha256.star
load("starcm", "download", "sha256")

# Hash a copy you trust once, then pin that hash in your configuration.
pinned = sha256("files/agent-1.2.0.json", label = "hash trusted copy").value
print(pinned)

r = download(
    label   = "fetch manifest",
    url     = "http://localhost:8000/docs/fixtures/download/files/agent-1.2.0.json",
    save_to = "manifest.json",
    sha256  = pinned,
)
print("success=%s changed=%s" % (r.success, r.changed))
```

```scrut
$ starcm pin_with_sha256.star
36f40179484cbb22f924bb9b88c8bf261363c078b3d58bc973038345072a11b5
success=True changed=True
```

### A wrong hash stops the run

The mismatching download is deleted, and nothing after the call runs.

```scrut
$ cat wrong_hash.star
load("starcm", "download")

download(
    label   = "fetch agent",
    url     = "http://localhost:8000/docs/fixtures/download/files/agent-1.2.0.txt",
    save_to = "tampered.txt",
    sha256  = "0000000000000000000000000000000000000000000000000000000000000000",
)
print("never reached")
```

```scrut
$ starcm wrong_hash.star 2>&1
Traceback (most recent call last):
  wrong_hash.star:3:9: in <toplevel>
Error in download: expected sha256 hash 0000000000000000000000000000000000000000000000000000000000000000, got ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08
[1]
```

```scrut
$ test ! -e tampered.txt && echo "tampered.txt was removed"
tampered.txt was removed
```

### Download into the local store

Leave out `save_to` and the file goes into the store. Here `--store-dir`
puts the store in the scratch directory.

The store examples use a store in the scratch directory. On Windows, starcm
refuses a store anywhere under the CI runners' `D:\` drive (where scrut's
scratch directory lives), because the drive is owned by another account, so
there the store goes under `%LocalAppData%`:

```scrut
$ store="$TMPDIR/store"; case "$(uname -s)" in MINGW*|MSYS*|CYGWIN*) store_root=$(mktemp -d "$(cygpath -u "$LOCALAPPDATA")/starcm-docs.XXXXXX") && store="$store_root/store" ;; esac
```

```scrut
$ cat store.star
load("starcm", "download")

r = download(
    label  = "fetch agent into the store",
    url    = "http://localhost:8000/docs/fixtures/download/files/agent-1.2.0.txt",
    sha256 = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
)
print("changed=%s" % r.changed)
print(r.value.replace("\\", "/"))  # / on every platform, for this example
```

```scrut
$ starcm --store-dir "$store" store.star
changed=True
*/store/sha256/ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08/agent-1.2.0.txt (glob)
```

The next run finds it in the store, checks its hash again, and returns the
same path:

```scrut
$ starcm --store-dir "$store" store.star
changed=False
*/store/sha256/ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08/agent-1.2.0.txt (glob)
```

```scrut
$ cat "$store/sha256/ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08/agent-1.2.0.txt"
agent v1.2.0
```

### Choose the stored file's name

The name comes from the URL unless `file_name` is given:

```scrut
$ cat store_named.star
load("starcm", "download")

r = download(
    label     = "fetch agent under another name",
    url       = "http://localhost:8000/docs/fixtures/download/files/agent-1.2.0.txt",
    sha256    = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
    file_name = "agent.txt",
)
print(r.value.replace("\\", "/"))
```

```scrut
$ starcm --store-dir "$store" store_named.star
*/store/sha256/ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08/agent.txt (glob)
```

### Require the store with `--store-only`

```scrut
$ cat store_only.star
load("starcm", "download")

download(
    label   = "fetch agent to a path",
    url     = "http://localhost:8000/docs/fixtures/download/files/agent-1.2.0.txt",
    save_to = "agent-1.2.0.txt",
    sha256  = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
)
```

```scrut
$ starcm --store-only --store-dir "$store" store_only.star 2>&1
Traceback (most recent call last):
  store_only.star:3:9: in <toplevel>
Error in download: download(label="fetch agent to a path"): save_to is disabled by --store-only; omit it to download into the store
[1]
```

```scrut
$ rm -rf "$store" ${store_root:+"$store_root"}
```

## See also

- [sha256](sha256.md) to compute the hash to pin.
- [exec](exec.md) re-checks stored files before running them.
- [verify](verify.md) to check a download's signature as well as its hash.
- [tests/download.md](../../tests/download.md) for HTTP errors, changed stored files and other edge cases.
