# load_remote

[← All functions](../README.md)

Use `load_remote` to fetch a `.star` file from a web server or an S3 bucket
and load it in one step. It does what [download](download.md) (or
[s3_download](s3_download.md)) followed by [load_dynamic](load_dynamic.md)
does, and returns the module's globals.

The file always goes into the [local store](download.md#the-local-store),
in a folder named after its sha256, and is loaded from there. So the file
that runs is the one that matched its pin, and a second run doesn't fetch it
again. Pin it by `sha256` or, under
[--trusted-keys](../cli.md#--trusted-keys-dir), by `signature`.

A URL ending in `/` loads a [folder](#folders) instead: every file its
signed manifest lists, laid out as a folder in the store, so the module can
`load()` its neighbours and use its templates and scripts the way it does
from a checkout.

## Syntax

```python
load("starcm", "load_remote")

load_remote("https://config.example.com/screen_lock.star", sha256 = "<hex>", label = "screen lock")
load_remote(
    url       = "https://config.example.com/screen_lock.star",
    signature = "https://config.example.com/screen_lock.star.sig",
    label     = "screen lock",
)
load_remote(
    url       = "https://config.example.com/agent/",   # a folder
    entry     = "main.star",
    signature = "https://config.example.com/agent/starcm-manifest.json.sig",
    label     = "agent config",
)
load_remote(
    url       = "s3://config/screen_lock.star",
    signature = "s3://config/screen_lock.star.sig",
    endpoint  = vars["s3_endpoint"],  # optional, for S3-compatible stores
    label     = "screen lock",
)
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `url` | string | required | The module, positionally or as `url =`: an `http://` or `https://` URL, or `s3://bucket/key`. Its name must end in `.star`, or the URL must end in `/` to load a [folder](#folders). |
| `entry` | string | `"main.star"` | Folders only: the module to run, as a path in the folder's manifest, such as `"site/main.star"`. |
| `sha256` | string | none | The sha256 the file must have; for a folder, the sha256 of its `starcm-manifest.json`. |
| `signature` | string | none | The URL of the file's signature, checked against `--trusted-keys`; for a folder, the manifest's (usually `<url>starcm-manifest.json.sig`). For `s3://` URLs, an `s3://` URL in the same bucket. See [pinning by signature](download.md#pinning-by-signature). |
| `endpoint` | string | AWS | `s3://` only: the S3-compatible server, as for [s3_download](s3_download.md). |
| `region` | string | AWS configuration, else `us-east-1` | `s3://` only: the bucket's region. |

Give exactly one of `sha256` and `signature`. S3 credentials come from the
AWS SDK's usual sources, as for [s3_download](s3_download.md).

`load_remote` also takes the [common arguments](../common-functionality.md#common-arguments):
`label` (required), `only_if`, `not_if`, `timeout` and
`what_if`. `timeout` limits the download.

## Returns

A [starcm_result](../common-functionality.md#starcm_result):

| Field | Value |
| --- | --- |
| `success` | `True` |
| `changed` | `True` when the file (for a folder, any of its files), or its signature, was fetched into the store |
| `message` | `loaded <url> from <path in the store>`; for a folder, `loaded <url><entry> from <path in the store>` |
| `value` | A frozen dict of the module's globals: `value["NAME"]` |

A file that doesn't match its pin, a refused signature, a failed download or
an error while the module runs stops the run.

## Loading in the store

The module runs from its folder in the store, so:

- **`--trusted-keys`:** the module needs a signature like any other `.star`
  file. Pin it by `signature`: the signature is kept next to the stored file,
  where the loader finds it. A module pinned by `sha256` is refused.
- **Once per run:** loading the same file again returns the same globals
  without running it again.
- **Other files:** a single remote module's folder in the store only holds
  the module, so nothing ships next to it. Its relative `load()` statements
  look there, and a relative bundled-resource path in it, such as
  `template`'s `template`, or [here()](here.md), is an error. Publish the
  module and its files as a [folder](#folders), have it fetch the files it
  needs with `load_remote` or `download`, or take their paths as arguments.

## Folders

A URL ending in `/` names a folder signed with
[starcm sign --dir](../cli.md#starcm-sign---dir-dir), which wrote a
`starcm-manifest.json` listing every file in it with its sha256 and a
signature for the manifest. `load_remote` fetches it the way
[download](download.md#folders) does, then runs `entry` from it:

1. It fetches the manifest and checks it against `sha256` or, under
   `--trusted-keys`, its signature.
2. It fetches each listed file relative to `url` and checks it against the
   manifest. A missing file or a wrong hash stops the run, and files the
   manifest doesn't list are never fetched.
3. It lays the files out at `<store>/folders/<manifest sha256>/`, re-checks
   the folder against the manifest, and runs `entry` there.

Inside the folder:

- **Paths work as in a checkout.** Relative `load()` statements, `here()` and
  relative bundled-resource paths resolve next to the `.star` file, within
  the folder. They can't leave it: `..` past its root is an error.
- **One signature covers the folder.** Under `--trusted-keys`, a `.star`
  file in the folder is trusted when the manifest's signature verifies and
  the file has the sha256 the manifest lists; it needs no `.sig` of its own.
  Pin the folder by `signature`, since a folder pinned by `sha256` has no
  signature to check.
- **Re-checked:** every module is checked against the manifest right before
  it runs, and [exec](exec.md) re-checks a script from the folder before
  running it. A file changed after it was stored stops the run.
- **Once per run**, as for one file: loading the same folder and `entry`
  again returns the same globals.

A new release is a new upload of the folder, its manifest and signature
last; only the files whose hashes changed are downloaded.

## Examples

Each example runs in a scratch copy of
[examples/load_remote/](../../examples/load_remote). The module they load,
`remote/screen_lock.star`, is served from this repository by
`python3 -m http.server 8000`. `$store` is a scratch store (on Windows, under
`%LocalAppData%`, since starcm refuses a store on a drive other accounts can
change).

```scrut
$ cp -R "$TESTDIR/../../examples/load_remote/." "$TMPDIR" && cd "$TMPDIR"
```

```scrut
$ store="$TMPDIR/store"; case "$(uname -s)" in MINGW*|MSYS*|CYGWIN*) store_root=$(mktemp -d "$(cygpath -u "$LOCALAPPDATA")/starcm-docs.XXXXXX") && store="$store_root/store" ;; esac
```

```python
# examples/load_remote/remote/screen_lock.star
load("starcm", "write")

SCREEN_LOCK_SECONDS = 300

def configure():
    write("require a password %d seconds after the screen saver starts" % SCREEN_LOCK_SECONDS, label = "screen lock")
```

### Load a module pinned by sha256

```python
# examples/load_remote/pinned.star
load("starcm", "load_remote")

r = load_remote(
    "http://localhost:8000/examples/load_remote/remote/screen_lock.star",
    sha256 = "3ad71837f20bade1d1eb37f5552d173bd4e03551eed67a829a2cd2173640d983",
    label  = "screen lock settings",
)
print("changed=%s" % r.changed)
r.value["configure"]()
```

```scrut
$ starcm --store-dir "$store" pinned.star
changed=True
require a password 300 seconds after the screen saver starts
```

The second run loads it from the store without fetching it:

```scrut
$ starcm --store-dir "$store" pinned.star
changed=False
require a password 300 seconds after the screen saver starts
```

```scrut
$ ls "$store"/sha256/3ad71837f20bade1d1eb37f5552d173bd4e03551eed67a829a2cd2173640d983/
screen_lock.star
```

### Load a signed module with `--trusted-keys`

The module's signature,
[remote/screen_lock.star.sig](../../examples/load_remote/remote/screen_lock.star.sig),
was made with the test-only key from [docs/keys/](../keys) and is served next
to it:

```python
# examples/load_remote/signed.star
load("starcm", "load_remote")

# No sha256 here: the signature, checked against --trusted-keys, says which
# sha256 the module must have.
settings = load_remote(
    url       = "http://localhost:8000/examples/load_remote/remote/screen_lock.star",
    signature = "http://localhost:8000/examples/load_remote/remote/screen_lock.star.sig",
    label     = "screen lock settings",
).value
settings["configure"]()
```

`signed.star` runs under `--trusted-keys`, so it's signed too:

```scrut
$ mkdir trusted && cp "$TESTDIR/../keys/trusted/test.pub.pem" trusted/ && starcm sign --expires 30d --key "$TESTDIR/../keys/private/test.pem" signed.star pinned.star >/dev/null && starcm --trusted-keys trusted --store-dir "$store" signed.star
require a password 300 seconds after the screen saver starts
```

The signature is stored next to the module:

```scrut
$ ls "$store"/sha256/3ad71837f20bade1d1eb37f5552d173bd4e03551eed67a829a2cd2173640d983/
screen_lock.star
screen_lock.star.sig
```

### Load a folder

[examples/folders/agent/](../../examples/folders/agent) is a folder signed
with `starcm sign --dir`. Its `main.star` loads `lib/settings.star` and
renders `templates/motd.jinja`, both by relative path:

```python
# examples/folders/agent/main.star
load("starcm", "template", "vars")
load("lib/settings.star", "SETTINGS")

def configure():
    """Writes the message of the day from the template in this folder."""
    r = template(
        label       = "motd",
        template    = "templates/motd.jinja",  # ships in this folder
        data        = SETTINGS,
        destination = vars["managed_dir"] + "/motd",
    )
    print("motd: changed=%s" % r.changed)
```

```python
# examples/folders/agent/lib/settings.star
SETTINGS = {
    "channel": "stable",
    "server":  "https://config.example.com",
}
```

```jinja
{# examples/folders/agent/templates/motd.jinja #}
Managed by starcm from {{ server }} ({{ channel }} channel).
```

`examples/folders/agent/starcm-manifest.json`:

```json
{
  "version": 1,
  "files": {
    "lib/settings.star": "2fd0f55fc725e6c4d0844f3e5f3be1a58d4cb62b43673ac0184bbefb04a900fc",
    "main.star": "4554ac7ab8fa570d4b4eb5aa85202e4eb8f20687585387228c93cd1ac4380a5f",
    "templates/motd.jinja": "e6a92af32114f46a9b524506f7ecf36aa75c3157ce3663573a568650a41a609d"
  }
}
```

One signature, on the manifest, loads all of it:

```python
# examples/load_remote/folder.star
load("starcm", "load_remote")

# A URL ending in / loads a folder. The signature covers the folder's
# manifest, which pins every file in it, so main.star, lib/settings.star and
# templates/motd.jinja need no .sig of their own. entry is the module to run.
agent = load_remote(
    url       = "http://localhost:8000/examples/folders/agent/",
    entry     = "main.star",
    signature = "http://localhost:8000/examples/folders/agent/starcm-manifest.json.sig",
    label     = "agent config",
).value
agent["configure"]()
```

```scrut
$ starcm sign --expires 30d --key "$TESTDIR/../keys/private/test.pem" folder.star >/dev/null && starcm --trusted-keys trusted --store-dir "$store" --var managed_dir="$(pwd -W 2>/dev/null || pwd)" folder.star && cat motd
motd: changed=True
Managed by starcm from https://config.example.com (stable channel).
```

The folder is in the store under its manifest's sha256, with the
manifest's signature next to the manifest:

```scrut
$ ls "$store"/folders/f7e8fbe643ea525dce9c8e0138f2ff4cad6d5ab1d539ed7a35ac55be380981ca/
lib
main.star
starcm-manifest.json
starcm-manifest.json.sig
templates
```

### Load a module from S3

[tests/integration/s3/chain.md](../../tests/integration/s3/chain.md) runs a
chain of signed modules, and a signed folder, loaded with `load_remote` from
an S3-compatible server.

## Errors and edge cases

These are here mostly as tests: CI runs them to check error messages and
behaviour that the examples above don't show. They run in the copy of
[examples/load_remote/edge/](../../examples/load_remote/edge).

### A module pinned by sha256 is refused under --trusted-keys

With an empty store there's no signature to put next to the module:

```scrut
$ rm -rf "$store" && starcm --trusted-keys trusted --store-dir "$store" pinned.star 2>&1
Traceback (most recent call last):
  pinned.star:3:16: in <toplevel>
Error in load_remote: signature check failed for *screen_lock.star: no signature file *screen_lock.star.sig (glob)
[1]
```

```scrut
$ cd edge
```

### A module loads once per run

```python
# examples/load_remote/edge/hello.star
print("hello.star ran")

GREETING = "hello"
```

```python
# examples/load_remote/edge/twice.star
load("starcm", "load_remote")

# The same module twice in one run: it runs once, and both calls get its globals.
for i in [1, 2]:
    r = load_remote("http://localhost:8000/examples/load_remote/edge/hello.star", sha256 = "d5060e7e366df6090f58f576d6c499df6e44ba7d8b334e68a76dcf1d02b32b03", label = "hello %d" % i)
    print("%d: %s" % (i, r.value["GREETING"]))
```

```scrut
$ starcm --store-dir "$store" twice.star
hello.star ran
1: hello
2: hello
```

### Pins

```python
# examples/load_remote/edge/wrong_hash.star
load("starcm", "load_remote")

load_remote("http://localhost:8000/examples/load_remote/edge/hello.star", sha256 = "0000000000000000000000000000000000000000000000000000000000000000", label = "wrong hash")
```

```scrut
$ starcm --store-dir "$store" wrong_hash.star 2>&1
Traceback (most recent call last):
  wrong_hash.star:3:12: in <toplevel>
Error in load_remote: expected sha256 hash 0000000000000000000000000000000000000000000000000000000000000000, got d5060e7e366df6090f58f576d6c499df6e44ba7d8b334e68a76dcf1d02b32b03
[1]
```

```python
# examples/load_remote/edge/no_pin.star
load("starcm", "load_remote")

load_remote("http://localhost:8000/examples/load_remote/edge/hello.star", label = "no pin")
```

```scrut
$ starcm --store-dir "$store" no_pin.star 2>&1
Traceback (most recent call last):
  no_pin.star:3:12: in <toplevel>
Error in load_remote: load_remote(label="no pin"): sha256 or signature must be provided
[1]
```

### Relative resource paths in a remote module

A relative `template` path means "next to this `.star` file", which for a
remote module is its folder in the store. The error says so instead of
reporting a missing file:

```python
# examples/load_remote/edge/render.star
load("starcm", "template")

def render(dest):
    # motd.jinja would have to ship next to this file, but in the store
    # nothing does.
    template(label = "motd", template = "motd.jinja", data = {}, destination = dest)
```

```python
# examples/load_remote/edge/relative_resource.star
load("starcm", "here", "load_remote")

r = load_remote("http://localhost:8000/examples/load_remote/edge/render.star", sha256 = "fb8592c5b4d09d832ed6a84eb3e010c378b32141cf5e1d9913363dc9e86dff2e", label = "renderer")
r.value["render"](here("motd.txt"))
```

```scrut
$ starcm --store-dir "$store" relative_resource.star 2>&1
Traceback (most recent call last):
  relative_resource.star:4:18: in <toplevel>
  *render.star:6:13: in render (glob)
Error in template: template(label="motd"): template: "motd.jinja" is relative, but this .star file was loaded with load_remote and runs from starcm's store, where nothing ships next to it: fetch the file with download() or pass its absolute path
[1]
```

### URLs

```python
# examples/load_remote/edge/not_star.star
load("starcm", "load_remote")

load_remote("http://localhost:8000/examples/load_remote/edge/hello.txt", sha256 = "d5060e7e366df6090f58f576d6c499df6e44ba7d8b334e68a76dcf1d02b32b03", label = "not a .star file")
```

```scrut
$ starcm --store-dir "$store" not_star.star 2>&1
Traceback (most recent call last):
  not_star.star:3:12: in <toplevel>
Error in load_remote: load_remote(label="not a .star file"): http://localhost:8000/examples/load_remote/edge/hello.txt doesn't name a .star file
[1]
```

```python
# examples/load_remote/edge/bad_scheme.star
load("starcm", "load_remote")

load_remote("ftp://localhost/hello.star", sha256 = "d5060e7e366df6090f58f576d6c499df6e44ba7d8b334e68a76dcf1d02b32b03", label = "ftp")
```

```scrut
$ starcm --store-dir "$store" bad_scheme.star 2>&1
Traceback (most recent call last):
  bad_scheme.star:3:12: in <toplevel>
Error in load_remote: load_remote(label="ftp"): url must start with http://, https:// or s3://, got ftp://localhost/hello.star
[1]
```

```python
# examples/load_remote/edge/endpoint_http.star
load("starcm", "load_remote")

load_remote("http://localhost:8000/examples/load_remote/edge/hello.star", sha256 = "d5060e7e366df6090f58f576d6c499df6e44ba7d8b334e68a76dcf1d02b32b03", endpoint = "http://127.0.0.1:9000", label = "endpoint")
```

```scrut
$ starcm --store-dir "$store" endpoint_http.star 2>&1
Traceback (most recent call last):
  endpoint_http.star:3:12: in <toplevel>
Error in load_remote: load_remote(label="endpoint"): endpoint and region are only for s3:// URLs
[1]
```

An S3 module's signature must be in the same bucket:

```python
# examples/load_remote/edge/other_bucket.star
load("starcm", "load_remote")

load_remote("s3://config/main.star", signature = "s3://elsewhere/main.star.sig", label = "other bucket")
```

```scrut
$ starcm --store-dir "$store" other_bucket.star 2>&1
Traceback (most recent call last):
  other_bucket.star:3:12: in <toplevel>
Error in load_remote: load_remote(label="other bucket"): signature s3://elsewhere/main.star.sig must be an s3:// URL in bucket "config"
[1]
```

### Folders

A folder pinned by `sha256` runs without `--trusted-keys`:

```python
# examples/load_remote/edge/folder_pinned.star
load("starcm", "load_remote")

load_remote("http://localhost:8000/examples/folders/agent/", sha256 = "f7e8fbe643ea525dce9c8e0138f2ff4cad6d5ab1d539ed7a35ac55be380981ca", label = "agent config").value["configure"]()
```

```scrut
$ starcm --store-dir "$store" --var managed_dir="$(pwd -W 2>/dev/null || pwd)" folder_pinned.star
motd: changed=True
```

A file changed in the store after the folder was assembled stops the run:

```scrut
$ f="$store"/folders/f7e8fbe643ea525dce9c8e0138f2ff4cad6d5ab1d539ed7a35ac55be380981ca/lib/settings.star && chmod u+w "$f" && echo 'SETTINGS["channel"] = "beta"' >> "$f" && starcm --store-dir "$store" --var managed_dir="$(pwd -W 2>/dev/null || pwd)" folder_pinned.star 2>&1
Traceback (most recent call last):
  folder_pinned.star:3:12: in <toplevel>
Error in load_remote: folder *f7e8fbe643ea525dce9c8e0138f2ff4cad6d5ab1d539ed7a35ac55be380981ca in the store was changed after it was stored: lib/settings.star has sha256 *, but the manifest lists * (glob)
[1]
```

`entry` must be a `.star` file the manifest lists:

```python
# examples/load_remote/edge/folder_entry.star
load("starcm", "load_remote")

load_remote("http://localhost:8000/examples/folders/agent/", entry = "setup.star", sha256 = "f7e8fbe643ea525dce9c8e0138f2ff4cad6d5ab1d539ed7a35ac55be380981ca", label = "no such entry")
```

```scrut
$ rm -rf "$store" && starcm --store-dir "$store" folder_entry.star 2>&1
Traceback (most recent call last):
  folder_entry.star:3:12: in <toplevel>
Error in load_remote: load_remote(label="no such entry"): entry "setup.star" is not in http://localhost:8000/examples/folders/agent/starcm-manifest.json
[1]
```

and is only for folders:

```python
# examples/load_remote/edge/entry_file.star
load("starcm", "load_remote")

load_remote("http://localhost:8000/examples/load_remote/edge/hello.star", entry = "main.star", sha256 = "d5060e7e366df6090f58f576d6c499df6e44ba7d8b334e68a76dcf1d02b32b03", label = "entry for a file")
```

```scrut
$ starcm --store-dir "$store" entry_file.star 2>&1
Traceback (most recent call last):
  entry_file.star:3:12: in <toplevel>
Error in load_remote: load_remote(label="entry for a file"): entry is only for a folder: a url ending in /
[1]
```

Paths inside a folder can't leave it.
[edge/escape/main.star](../../examples/load_remote/edge/escape/main.star)
asks for a file two levels above its folder:

```python
# examples/load_remote/edge/escape/main.star
load("starcm", "here")

# Inside a fetched folder, here() and relative paths can't leave it.
print(here("..", "..", "secrets.txt"))
```

```python
# examples/load_remote/edge/folder_escape.star
load("starcm", "load_remote")

load_remote("http://localhost:8000/examples/load_remote/edge/escape/", sha256 = "ea2126eca18f7445344e8d4387487dbbce4ed4effcabb9b5a4cd0891a78e5e02", label = "escape")
```

```scrut
$ starcm --store-dir "$store" folder_escape.star 2>&1
Traceback (most recent call last):
  folder_escape.star:3:12: in <toplevel>
Error in load_remote: load failed:
Traceback (most recent call last):
  *main.star:4:11: in <toplevel> (glob)
Error in here: here: "../../secrets.txt" is outside the folder * that this .star file was fetched in with load_remote (glob)
[1]
```

```scrut
$ rm -rf "$store" ${store_root:+"$store_root"}
```

## See also

- [load_dynamic](load_dynamic.md) for `.star` files that are already on the machine.
- [download](download.md#pinning-by-signature) for how pins and signatures work.
- [inspect](inspect.md) reports each `load_remote` call's URL, or `null` when it's computed.
