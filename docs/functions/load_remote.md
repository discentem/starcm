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
    url       = "s3://config/screen_lock.star",
    signature = "s3://config/screen_lock.star.sig",
    endpoint  = vars["s3_endpoint"],  # optional, for S3-compatible stores
    label     = "screen lock",
)
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `url` | string | required | The module, positionally or as `url =`: an `http://` or `https://` URL, or `s3://bucket/key`. Its name must end in `.star`. |
| `sha256` | string | none | The sha256 the file must have. |
| `signature` | string | none | The URL of the file's signature, checked against `--trusted-keys`. For `s3://` URLs, an `s3://` URL in the same bucket. See [pinning by signature](download.md#pinning-by-signature). |
| `endpoint` | string | AWS | `s3://` only: the S3-compatible server, as for [s3_download](s3_download.md). |
| `region` | string | AWS configuration, else `us-east-1` | `s3://` only: the bucket's region. |

Give exactly one of `sha256` and `signature`. S3 credentials come from the
AWS SDK's usual sources, as for [s3_download](s3_download.md).

`load_remote` also takes the [common arguments](../common-functionality.md#common-arguments):
`label` (required), `only_if`, `not_if`, `timeout`, `working_directory` and
`what_if`. `timeout` limits the download.

## Returns

A [starcm_result](../common-functionality.md#starcm_result):

| Field | Value |
| --- | --- |
| `success` | `True` |
| `changed` | `True` when the file, or its signature, was fetched into the store |
| `message` | `loaded <url> from <path in the store>` |
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
- **Other files:** a remote module's relative `load()` statements look in its
  folder in the store, which only holds the module. Have it fetch the files
  it needs with `load_remote` or `download` instead.

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

### Load a module from S3

[tests/integration/s3/chain.md](../../tests/integration/s3/chain.md) runs a
chain of signed modules loaded with `load_remote` from an S3-compatible
server.

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

```scrut
$ rm -rf "$store" ${store_root:+"$store_root"}
```

## See also

- [load_dynamic](load_dynamic.md) for `.star` files that are already on the machine.
- [download](download.md#pinning-by-signature) for how pins and signatures work.
- [inspect](inspect.md) reports each `load_remote` call's URL, or `null` when it's computed.
