# download

[← All functions](../README.md)

Use `download` to fetch a file over HTTP(S) and check it against a pinned
sha256. If a file with the right hash is already at `save_to`, nothing is
downloaded, so running the same configuration again is cheap and reports
`changed = False`.

## Syntax

```python
load("starcm", "download")

download(
    label         = "fetch agent",       # required
    url           = "https://...",       # required
    save_to       = "agent.pkg",         # required
    sha256        = "ffcf2b6c...",       # required
    live_progress = False,
)
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `url` | string | required | URL to fetch. Any status other than `200 OK` fails the run. |
| `save_to` | string | required | Where to save the file, relative to the directory starcm runs in. Missing parent directories are created. |
| `sha256` | string | required | Expected lowercase hex sha256 of the file. Use [sha256()](sha256.md) or `shasum -a 256` to get it. |
| `live_progress` | bool | `False` | Print a `Downloading <url>... N%` progress line while downloading. |

`download` also takes the [common arguments](../common-functionality.md#common-arguments):
`label` (required), `only_if`, `not_if`, `timeout`, `working_directory` and
`what_if`.

## What it does

1. If `save_to` exists and its sha256 matches, return `changed = False`.
2. If it exists with a different hash, delete it.
3. Download `url` to `save_to`, hashing as it goes.
4. If the hash doesn't match `sha256`, delete the download and fail the run.

## Returns

A [starcm_result](../common-functionality.md#starcm_result):

| Field | Value |
| --- | --- |
| `success` | `True` |
| `changed` | `True` when the file was downloaded, `False` when it was already present |
| `message` | `downloaded file to <save_to>` or `"<save_to>" already present and sha256 verified` |
| `value` | `None` |

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

## See also

- [sha256](sha256.md) to compute the hash to pin.
- [verify](verify.md) to check a download's signature as well as its hash.
- [tests/download.md](../../tests/download.md) for HTTP errors and other edge cases.
