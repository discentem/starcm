# s3_download

[← All functions](../README.md)

Use `s3_download` to fetch an object from Amazon S3, or from an S3-compatible
store such as MinIO, Cloudflare R2, Wasabi or RustFS, using AWS credentials.
It checks the file against a pinned sha256 and saves it the same way
[download](download.md) does: to a path you choose with `save_to`, or into
starcm's [local store](download.md#the-local-store). If the file is already
there with the right hash, nothing is downloaded and `changed = False`.

It's separate from `download` so that [inspect](inspect.md) can tell the two
apart. A policy can then allow plain downloads but not authenticated ones, or
the other way round.

## Syntax

```python
load("starcm", "s3_download")

s3_download(
    label         = "fetch agent",                 # required
    bucket        = "releases",                    # required
    key           = "agent/agent-1.2.0.pkg",       # required
    sha256        = "ffcf2b6c...",                 # sha256 or signature is required
    signature     = "agent/agent-1.2.0.pkg.sig",   # key of the signature, same bucket
    save_to       = "/opt/agent/agent.pkg",        # omit to use the local store
    file_name     = "agent.pkg",                   # name in the store; only without save_to
    region        = "eu-west-1",
    endpoint      = "https://minio.example.com",   # S3-compatible stores only
    live_progress = False,
)
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `bucket` | string | required | The bucket. |
| `key` | string | required | The object's key, such as `agent/agent-1.2.0.pkg`. A key ending in `/` needs `folder = True`. |
| `folder` | bool | `False` | `key`, ending in `/` such as `agent/`, is a [folder](#folders)'s prefix. |
| `sha256` | string | `sha256` or `signature` | Expected lowercase hex sha256 of the file. Use [sha256()](sha256.md) or `shasum -a 256` to get it. |
| `signature` | string | `sha256` or `signature` | Key of the file's signature in the same bucket, made with [sign](sign.md). The file must have the sha256 it records. Needs [--trusted-keys](../cli.md#--trusted-keys-dir). See [Pinning by signature](download.md#pinning-by-signature) for when to use which. |
| `save_to` | string | none | Where to save the file, as an absolute path (`~` expands). Leave it out to download into the [local store](download.md#the-local-store). Refused under [--store-only](../cli.md#--store-dir-dir-and---store-only). |
| `file_name` | string | the last part of `key` | The file's name in the local store. Only used without `save_to`, and an error for a folder. |
| `region` | string | from the AWS configuration, else `us-east-1` | The bucket's region. |
| `endpoint` | string | AWS S3 | The endpoint of an S3-compatible store. Requests then use path-style addressing (`<endpoint>/<bucket>/<key>`). |
| `live_progress` | bool | `False` | Print a progress line while downloading. |

`s3_download` also takes the
[common arguments](../common-functionality.md#common-arguments): `label`
(required), `only_if`, `not_if`, `timeout` and
`what_if`.

## Credentials

Credentials never go in the `.star` file. starcm signs each request (AWS
Signature Version 4) with credentials from the AWS SDK's standard chain,
checked in this order:

1. The `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables,
   plus `AWS_SESSION_TOKEN` for temporary credentials.
2. The shared credentials and config files (`~/.aws/credentials`,
   `~/.aws/config`), using the `default` profile or the one named by
   `AWS_PROFILE`. A profile can use SSO, or run a `credential_process`
   command that hands out short-lived credentials, so no long-lived key needs
   to sit on the machine.
3. On an AWS server, the machine's instance role. That's why, when no
   credentials are found, the error mentions `EC2 IMDS`.

`region` and `endpoint` aren't secret, so it's fine to set them in the `.star`
file, or pass them in with [--var](vars.md). Without them, starcm uses the
AWS configuration (`AWS_REGION`, `AWS_ENDPOINT_URL_S3` or the profile). With
no region anywhere, it uses `us-east-1`. AWS refuses requests for buckets in
other regions, so set `region` for those.

## Folders

With `folder = True`, `key`, which must end in `/`, is a folder signed with
[starcm sign --dir](../cli.md#starcm-sign---dir-dir) and uploaded as it is,
for example with `aws s3 sync agent/ s3://config/agent/`. It works as for
[download](download.md#folders): `sha256` or `signature` (here the key of
`<key>starcm-manifest.json.sig`) pins the folder's manifest, each file the
manifest lists is fetched from `<key><path>` and checked against it, and the
folder goes into the store, or under `save_to`. No `ListObjects` permission
is needed: the manifest says what to fetch.

```python
s3_download(
    label     = "fetch agent files",
    bucket    = "config",
    key       = "agent/",
    folder    = True,
    signature = "agent/starcm-manifest.json.sig",
    save_to   = vars["managed_dir"] + "/agent",
)
```

## Returns

A [starcm_result](../common-functionality.md#starcm_result), the same as
[download](download.md#returns)'s:

| Field | Value |
| --- | --- |
| `success` | `True` |
| `changed` | `True` when the file was downloaded, `False` when it was already present |
| `message` | `downloaded file to <path>`, `"<save_to>" already present and sha256 verified` or `"<path>" already in the store and sha256 verified` |
| `value` | The file's absolute path: `save_to`, or its place in the store. |

Missing credentials, a refused request, a missing object and a sha256
mismatch all stop the run with an error.

## Examples

Here CI's local HTTP server stands in for an S3-compatible store, through
`endpoint`. The bucket `examples` and key `download/files/agent-1.2.0.txt`
become the path `/examples/download/files/agent-1.2.0.txt` on that server. It doesn't check credentials, but starcm still signs the request with
the ones in the environment. The example runs in a scratch copy of
[examples/s3_download/](../../examples/s3_download).

```scrut
$ cp -R "$TESTDIR/../../examples/s3_download/." "$TMPDIR" && cd "$TMPDIR" && managed="$(pwd -W 2>/dev/null || pwd)"
```

### Download from an S3-compatible store

```python
# examples/s3_download/fetch.star
load("starcm", "s3_download", "vars")

r = s3_download(
    label    = "fetch agent from S3",
    bucket   = "examples",
    key      = "download/files/agent-1.2.0.txt",
    sha256   = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
    save_to  = vars["managed_dir"] + "/agent-1.2.0.txt",
    # An S3-compatible store; leave this out for AWS S3.
    endpoint = "http://localhost:8000",
)
print(r.message)
```

```scrut
$ AWS_ACCESS_KEY_ID=example AWS_SECRET_ACCESS_KEY=example starcm --var managed_dir="$managed" fetch.star && cat agent-1.2.0.txt
downloaded file to *agent-1.2.0.txt (glob)
agent v1.2.0
```

The second run finds the file with the right hash and doesn't fetch it again:

```scrut
$ AWS_ACCESS_KEY_ID=example AWS_SECRET_ACCESS_KEY=example starcm --var managed_dir="$managed" fetch.star
"*agent-1.2.0.txt" already present and sha256 verified (glob)
```

### Pin by signature

The signature is fetched from the same bucket and checked against
`--trusted-keys` first, and the file must then have the sha256 it records.
The signature is saved next to the file.

```scrut
$ mkdir -p trusted && cp "$TESTDIR/../keys/trusted/test.pub.pem" trusted/ && starcm sign --expires 2099-01-01 --key "$TESTDIR/../keys/private/test.pem" signed.star >/dev/null
```

```python
# examples/s3_download/signed.star
load("starcm", "s3_download", "vars")

# The signature is another object in the same bucket.
r = s3_download(
    label     = "fetch signed agent from S3",
    bucket    = "examples",
    key       = "download/files/agent-1.2.0.txt",
    signature = "download/files/agent-1.2.0.txt.sig",
    save_to   = vars["managed_dir"] + "/signed/agent.txt",
    endpoint  = "http://localhost:8000",
)
print(r.message)
```

```scrut
$ AWS_ACCESS_KEY_ID=example AWS_SECRET_ACCESS_KEY=example starcm --var managed_dir="$managed" --trusted-keys trusted signed.star && ls signed
downloaded file to *agent.txt (glob)
agent.txt
agent.txt.sig
```

### Download a folder

The same server holds the signed folder
[examples/folders/agent/](../../examples/folders/agent), as the prefix
`folders/agent/` of the bucket `examples`:

```python
# examples/s3_download/folder.star
load("starcm", "s3_download", "vars")

# folder = True fetches a folder; its key ends in /. signature is the key of its manifest's
# signature, in the same bucket; the manifest pins every file in the folder.
r = s3_download(
    label     = "fetch agent files from S3",
    bucket    = "examples",
    key       = "folders/agent/",
    folder    = True,
    signature = "folders/agent/starcm-manifest.json.sig",
    save_to   = vars["managed_dir"] + "/agent",
    endpoint  = "http://localhost:8000",
)
print(r.message)
```

```scrut
$ starcm sign --expires 2099-01-01 --key "$TESTDIR/../keys/private/test.pem" folder.star >/dev/null && AWS_ACCESS_KEY_ID=example AWS_SECRET_ACCESS_KEY=example starcm --var managed_dir="$managed" --trusted-keys trusted folder.star && ls agent
downloaded 3 of 3 files to *agent (glob)
lib
main.star
starcm-manifest.json
starcm-manifest.json.sig
templates
```

### Download from AWS S3

Leave out `endpoint`, and set `region` if the bucket isn't in `us-east-1`.
Credentials come from the environment or an `~/.aws` profile, as above:

```python
load("starcm", "s3_download")

s3_download(
    label  = "fetch agent",
    bucket = "example-releases",
    key    = "agent/agent-1.2.0.pkg",
    sha256 = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
    region = "eu-west-1",
)
```

## See also

- [download](download.md) for HTTP(S), and for how the local store works.
- [verify](verify.md) to check a download's signature as well as its hash.
- [install_mac_pkg](install_mac_pkg.md) to install a downloaded `.pkg`.
- [tests/integration/s3/chain.md](../../tests/integration/s3/chain.md) runs a chain of signed `.star` files from a real S3-compatible server, [rustfs](https://github.com/rustfs/rustfs), in CI: credentials the server refuses, a tampered bucket and missing objects.
