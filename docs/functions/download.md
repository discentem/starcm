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
    sha256        = "ffcf2b6c...",       # sha256 or signature is required
    signature     = "https://.../agent.pkg.sig",
    save_to       = "agent.pkg",         # omit to use the local store
    file_name     = "agent.pkg",         # name in the store; only without save_to
    live_progress = False,
)
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `url` | string | required | URL to fetch. Any status other than `200 OK` fails the run. |
| `sha256` | string | `sha256` or `signature` | Expected lowercase hex sha256 of the file. Use [sha256()](sha256.md) or `shasum -a 256` to get it. |
| `signature` | string | `sha256` or `signature` | URL of the file's signature, made with [sign](sign.md). The file must have the sha256 it records. Needs [--trusted-keys](../cli.md#--trusted-keys-dir). See [Pinning by signature](#pinning-by-signature). |
| `save_to` | string | none | Where to save the file, relative to the directory starcm runs in. Missing parent directories are created. Leave it out to download into the [local store](#the-local-store). Refused under [--store-only](../cli.md#--store-dir-dir-and---store-only). |
| `file_name` | string | the last part of the URL's path | The file's name in the local store. Only used without `save_to`. |
| `live_progress` | bool | `False` | Print a `Downloading <url>... N%` progress line while downloading. |

`download` also takes the [common arguments](../common-functionality.md#common-arguments):
`label` (required), `only_if`, `not_if`, `timeout`, `working_directory` and
`what_if`.

## Pinning by signature

Give exactly one of `sha256` or `signature`.

- **`sha256`** accepts exactly one build of the file. Each new release
  changes your configuration.
- **`signature`** accepts any file your release key has signed, as long as
  its signature hasn't expired. starcm first fetches the signature and
  checks it against [--trusted-keys](../cli.md#--trusted-keys-dir), including
  [--max-expiry](../cli.md#--max-expiry-duration). It then downloads the file
  as if the sha256 recorded in the signature had been given as `sha256`, so
  the second run still downloads nothing. Publishing a release means
  uploading the file and its new `.sig`, with no change to your
  configuration.

The signature is saved next to the file, as `<save_to>.sig` or next to the
file in the local store. So `verify()` finds it, and a `.star` file fetched
this way can be loaded with [load_dynamic](load_dynamic.md) under
`--trusted-keys`.

A pinned `sha256` refuses things a signature accepts:

- an older release that's still validly signed, if someone who can write to
  the server swaps it in. Short `--expires` and `--max-expiry` narrow that
  window;
- anything signed with a leaked release key, until the key is removed from
  `--trusted-keys`.

Use `sha256` when that matters more than not having to change your
configuration for each release.

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
  store's directory. By default it's
  `/Library/Application Support/starcm/store` (macOS), `/var/lib/starcm/store` (Linux) or
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
of [examples/download/](../../examples/download).

```scrut
$ cp -R "$TESTDIR/../../examples/download/." "$TMPDIR" && cd "$TMPDIR"
```

### Download a file and verify its hash

```python
# examples/download/basic.star
load("starcm", "download")

r = download(
    label   = "fetch agent",
    url     = "http://localhost:8000/examples/download/files/agent-1.2.0.txt",
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

```python
# examples/download/nested.star
load("starcm", "download")

r = download(
    label   = "fetch agent into a cache directory",
    url     = "http://localhost:8000/examples/download/files/agent-1.2.0.txt",
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

```python
# examples/download/from_var.star
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
$ rm agent-1.2.0.txt && starcm --var base_url=http://localhost:8000/examples/download/files from_var.star
fetch agent 1.2.0: success=True changed=True
```

### Pin a hash computed with `sha256()`

```python
# examples/download/pin_with_sha256.star
load("starcm", "download", "sha256")

# Hash a copy you trust once, then pin that hash in your configuration.
pinned = sha256("files/agent-1.2.0.json", label = "hash trusted copy").value
print(pinned)

r = download(
    label   = "fetch manifest",
    url     = "http://localhost:8000/examples/download/files/agent-1.2.0.json",
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

### Pin by signature

These examples use the test-only key from
[docs/keys/](../keys), and a signature
made with it,
[agent-1.2.0.txt.sig](../../examples/download/files/agent-1.2.0.txt.sig). Under
`--trusted-keys`, the `.star` file itself must be signed too.

```scrut
$ mkdir -p trusted && cp "$TESTDIR/../keys/trusted/test.pub.pem" trusted/ && starcm sign --expires 2099-01-01 --key "$TESTDIR/../keys/private/test.pem" signed.star >/dev/null
```

```python
# examples/download/signed.star
load("starcm", "download")

# No sha256 here: the signature, checked against --trusted-keys, says which
# sha256 the file must have.
r = download(
    label     = "fetch signed agent",
    url       = "http://localhost:8000/examples/download/files/agent-1.2.0.txt",
    signature = "http://localhost:8000/examples/download/files/agent-1.2.0.txt.sig",
    save_to   = "signed/agent.txt",
)
print(r.message)
```

```scrut
$ starcm --trusted-keys trusted signed.star && cat signed/agent.txt && ls signed
downloaded file to signed/agent.txt
agent v1.2.0
agent.txt
agent.txt.sig
```

Without `--trusted-keys` there's nothing to check the signature against:

```scrut
$ starcm signed.star 2>&1 | tail -1
Error in download: download(label="fetch signed agent"): signature needs trusted keys: run starcm with --trusted-keys
```

### A wrong hash stops the run

The mismatching download is deleted, and nothing after the call runs.

```python
# examples/download/wrong_hash.star
load("starcm", "download")

download(
    label   = "fetch agent",
    url     = "http://localhost:8000/examples/download/files/agent-1.2.0.txt",
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

```python
# examples/download/store.star
load("starcm", "download")

r = download(
    label  = "fetch agent into the store",
    url    = "http://localhost:8000/examples/download/files/agent-1.2.0.txt",
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

```python
# examples/download/store_named.star
load("starcm", "download")

r = download(
    label     = "fetch agent under another name",
    url       = "http://localhost:8000/examples/download/files/agent-1.2.0.txt",
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

```python
# examples/download/store_only.star
load("starcm", "download")

download(
    label   = "fetch agent to a path",
    url     = "http://localhost:8000/examples/download/files/agent-1.2.0.txt",
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

### Download from the internet

This one downloads a real release, Ghostty 1.2.3 for macOS (about 30 MB):

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

```scrut
$ starcm a_file.star
starcm_result(changed = True, error = "<nil>", label = "Downloading Ghostty 1.2.3", message = "downloaded file to Ghostty-1.2.3.dmg", success = True, value = None)
```

### Fetch a `.star` file from a server and run it

A small macro wraps `download` and [load_dynamic](load_dynamic.md):

```python
# examples/download/load_remote/load_remote_macro.star
load("starcm", "load_dynamic")
load("starcm", "download")

def load_remote(url, sha256):
    fname = url.split("/")[-1]
    d = download(
        label = "downloading {}".format(fname),
        url = url,
        save_to = fname,
        sha256 = sha256
    )
    load_dynamic(fname, label = "loading {}".format(fname))
```

```python
# examples/download/load_remote/load_remote.star
load("load_remote_macro.star", "load_remote")

# use the load_remote macro to download and load the remote file.
# Served by `python3 -m http.server 8000` from the repo root.
load_remote(
    url = "http://localhost:8000/examples/loading/nested_files/e.star",
    sha256 = "6a6afe8b43b89e345842f74d84cf92e280862d648f01f8f0fa356a9d4c6e8796"
)
```

The second run finds the file already downloaded, and runs it again:

```scrut
$ (cd load_remote && starcm load_remote.star && starcm load_remote.star)
hello from e.star
hello from e.star
```

## Errors and edge cases

These are here mostly as tests: CI runs them to check error messages and
behaviour that the examples above don't show. They run in the copy of
[examples/download/edge/](../../examples/download/edge).

```scrut
$ cd "$TMPDIR/edge"
```

```scrut
$ store="$TMPDIR/store"; case "$(uname -s)" in MINGW*|MSYS*|CYGWIN*) store_root=$(mktemp -d "$(cygpath -u "$LOCALAPPDATA")/starcm-scrut.XXXXXX") && store="$store_root/store" ;; esac
```

### Downloads once, then verifies the existing file

```python
# examples/download/edge/download.star
load("starcm", "download")

# Served by the python http.server that the scrut CI job runs from the repo root.
URL = "http://localhost:8000/examples/download/edge/payload.txt"
SHA256 = "77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314"

def show(r):
    print("%s: success=%s changed=%s message=%s" % (r.label, r.success, r.changed, r.message))

show(download(label = "first download", url = URL, save_to = "downloaded.txt", sha256 = SHA256))
show(download(label = "already present", url = URL, save_to = "downloaded.txt", sha256 = SHA256))
```

```scrut
$ starcm download.star
first download: success=True changed=True message=downloaded file to downloaded.txt
already present: success=True changed=False message="downloaded.txt" already present and sha256 verified
```

```scrut
$ cat downloaded.txt
starcm download fixture
```

The download is written to a temporary file and renamed into place once its
sha256 matches, so nothing is left behind next to it.

```scrut
$ ls -a | grep -c '\.tmp-'
0
[1]
```

### HTTP errors fail the run

```python
# examples/download/edge/not_found.star
load("starcm", "download")

download(
    label   = "404",
    url     = "http://localhost:8000/examples/download/edge/does-not-exist.txt",
    save_to = "nope.txt",
    sha256  = "0000000000000000000000000000000000000000000000000000000000000000",
)
```

```scrut
$ starcm not_found.star 2>&1
Traceback (most recent call last):
  not_found.star:3:9: in <toplevel>
Error in download: failed to download file "http://localhost:8000/examples/download/edge/does-not-exist.txt": 404 File not found
[1]
```

### Without save_to, downloads go into the store

Files are stored as `<store>/sha256/<hash>/<name>`, named after the URL unless
`file_name` is given, and `value` is the stored path. A stored file is
re-hashed whenever `download()` finds it already in the store, and whenever
`exec()` is given its path.

```python
# examples/download/edge/store.star
load("starcm", "download")

# Without save_to, download() saves into the store (--store-dir) and returns
# the stored file's path as value.
URL = "http://localhost:8000/examples/download/edge/payload.txt"
SHA256 = "77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314"

def show(r):
    print("%s: changed=%s value=%s" % (r.label, r.changed, r.value.replace("\\", "/")))

show(download(label = "into the store", url = URL, sha256 = SHA256))
show(download(label = "already stored", url = URL, sha256 = SHA256))
show(download(label = "named", url = URL, sha256 = SHA256, file_name = "renamed.txt"))
```

```scrut
$ starcm --store-dir "$store" store.star
into the store: changed=True value=*/store/sha256/77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314/payload.txt (glob)
already stored: changed=False value=*/store/sha256/77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314/payload.txt (glob)
named: changed=True value=*/store/sha256/77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314/renamed.txt (glob)
```

```scrut
$ cat "$store"/sha256/77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314/payload.txt
starcm download fixture
```

### A changed file in the store is refused

```scrut
$ f="$store"/sha256/77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314/payload.txt && chmod u+w "$f" && echo edited >> "$f" && starcm --store-dir "$store" store.star 2>&1
Traceback (most recent call last):
  store.star:11:14: in <toplevel>
Error in download: *payload.txt in the store has sha256 *; it was changed after it was stored (glob)
[1]
```

### --store-only refuses save_to

```python
# examples/download/edge/store_only.star
load("starcm", "download")

download(
    label   = "to a path",
    url     = "http://localhost:8000/examples/download/edge/payload.txt",
    save_to = "downloaded.txt",
    sha256  = "77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314",
)
```

```scrut
$ starcm --store-only store_only.star 2>&1
Traceback (most recent call last):
  store_only.star:3:9: in <toplevel>
Error in download: download(label="to a path"): save_to is disabled by --store-only; omit it to download into the store
[1]
```

```scrut
$ rm -rf "$store" && starcm --store-only --store-dir "$store" store.star | head -1
into the store: changed=True value=*/store/sha256/77343010f71b589c625b217d4a544a08dbcf49ab1ba1e99b369c4a6c378bf314/payload.txt (glob)
```

```scrut
$ rm -rf "$store" ${store_root:+"$store_root"}
```

## See also

- [sha256](sha256.md) to compute the hash to pin.
- [exec](exec.md) re-checks stored files before running them.
- [verify](verify.md) to check a download's signature as well as its hash.
