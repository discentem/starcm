# s3_download(): a signed fleet served from a real S3 server

Runs a starcm deployment end to end against a real S3-compatible server,
[rustfs](https://github.com/rustfs/rustfs), which checks every request's
signature. This runs only in the scrut CI workflow
(`.github/workflows/scrut.yml`). That workflow starts rustfs, then sets
`STARCM_TEST_S3_ENDPOINT` and test credentials in `AWS_ACCESS_KEY_ID` and
`AWS_SECRET_ACCESS_KEY`. It isn't part of the regular `tests/` run.

The machine starts with only `bootstrap.star`. That file fetches `main.star`
from the `fleet` bucket into the local store and runs it. `main.star` fetches
a role and a payload the same way, loads the role and runs the payload.
Everything runs under `--trusted-keys`. No file is pinned by sha256: each is
pinned by its signature, `<key>.sig` in the same bucket, so a new release
only needs new files and signatures in the bucket.

## Set up the bucket

Sign the fleet with the test key from `tests/fixtures/signing/`, upload it,
and remove the local copy, so everything has to come from the bucket.

```scrut
$ cp -R "$TESTDIR/fixtures/." "$TMPDIR" && cd "$TMPDIR" && mkdir trusted && cp "$TESTDIR/../../tests/fixtures/signing/trusted/test.pub.pem" trusted/ && key="$TESTDIR/../../tests/fixtures/signing/private/test.pem" && untrusted="$TESTDIR/../../tests/fixtures/signing/private/untrusted.pem"
```

```scrut
$ (cd fleet && starcm sign --expires 30d --key "$key" main.star roles/web.star payloads/agent.sh) && starcm sign --expires 30d --key "$key" bootstrap.star
signed main.star with key "test" until * -> main.star.sig (glob)
signed roles/web.star with key "test" until * -> roles/web.star.sig (glob)
signed payloads/agent.sh with key "test" until * -> payloads/agent.sh.sig (glob)
signed bootstrap.star with key "test" until * -> bootstrap.star.sig (glob)
```

```scrut
$ s3() { aws --endpoint-url "$STARCM_TEST_S3_ENDPOINT" s3 "$@"; } && s3 mb s3://fleet >/dev/null && s3 sync --quiet fleet s3://fleet && rm -rf fleet && s3 ls --recursive s3://fleet | awk '{print $4}' | sort
main.star
main.star.sig
payloads/agent.sh
payloads/agent.sh.sig
roles/web.star
roles/web.star.sig
```

Fetched files go into a local store. On the Windows CI runners, starcm
refuses a store under `D:\` (where `$TMPDIR` is) because another account owns
the drive, so there the store goes under `%LocalAppData%`.

```scrut
$ store="$TMPDIR/store"; case "$(uname -s)" in MINGW*|MSYS*|CYGWIN*) store_root=$(mktemp -d "$(cygpath -u "$LOCALAPPDATA")/starcm-s3.XXXXXX") && store="$store_root/store" ;; esac; run() { starcm --trusted-keys trusted --store-dir "$store" --var "s3_endpoint=$STARCM_TEST_S3_ENDPOINT" bootstrap.star; }
```

## The fleet runs

```scrut
$ run
web role configured
agent 1.0 installed
```

Nothing was fetched into the directory it ran in (the store sits there too
on macOS and Linux, so it's left out of the listing). Each fetched file is
in the store, with its signature next to it, which is how `--trusted-keys`
checked `main.star` and the role before loading them:

```scrut
$ ls | grep -vx store
bootstrap.star
bootstrap.star.sig
trusted
```

```scrut
$ find "$store" -type f | sed 's|.*[/\\]||' | sort
agent.sh
agent.sh.sig
main.star
main.star.sig
web.star
web.star.sig
```

A second run finds every file in the store with its signed sha256, so it
only fetches the signatures again:

```scrut
$ run
web role configured
agent 1.0 installed
```

## A new release needs no configuration change

Publish agent 2.0: upload the new payload and its new signature. Nothing on
the machine changes.

```scrut
$ printf '#!/bin/sh\necho "agent 2.0 installed"\n' > agent.sh && starcm sign --expires 30d --key "$key" agent.sh >/dev/null && s3 cp --quiet agent.sh s3://fleet/payloads/agent.sh && s3 cp --quiet agent.sh.sig s3://fleet/payloads/agent.sh.sig && run
web role configured
agent 2.0 installed
```

## A tampered bucket is refused

Someone with write access to the bucket swaps in a payload that the
signature doesn't cover. While the signed release is already in the store,
that one runs and the swap changes nothing:

```scrut
$ printf '#!/bin/sh\necho pwned\n' > evil.sh && s3 cp --quiet evil.sh s3://fleet/payloads/agent.sh && run
web role configured
agent 2.0 installed
```

On a machine that doesn't have it yet, the download doesn't match the sha256
the signature records, so it's refused before it's stored or run:

```scrut
$ rm -rf "$store" && run 2>&1 | tail -1
Error in s3_download: expected sha256 hash *, got * (glob)
```

Signing the swap with a key the machine doesn't trust doesn't help:

```scrut
$ starcm sign --expires 30d --key "$untrusted" evil.sh >/dev/null && s3 cp --quiet evil.sh.sig s3://fleet/payloads/agent.sh.sig && run 2>&1 | tail -1
Error in s3_download: s3_download(label="fetch payloads/agent.sh"): signature payloads/agent.sh.sig: key "untrusted" is not trusted
```

## The server checks credentials

```scrut
$ AWS_SECRET_ACCESS_KEY=wrong run 2>&1 | tail -1
Error in s3_download: s3_download(label="fetch main.star"): fetching signature: failed to download s3://fleet/main.star.sig: operation error S3: GetObject, https response error StatusCode: 403, RequestID: *, HostID: , api error SignatureDoesNotMatch: * (glob)
```

```scrut
$ AWS_ACCESS_KEY_ID=nobody run 2>&1 | tail -1
Error in s3_download: s3_download(label="fetch main.star"): fetching signature: failed to download s3://fleet/main.star.sig: operation error S3: GetObject, https response error StatusCode: 403, RequestID: *, HostID: , api error InvalidAccessKeyId: * (glob)
```

## A missing object fails the run

```scrut
$ s3 rm --quiet s3://fleet/main.star.sig && run 2>&1 | tail -1
Error in s3_download: s3_download(label="fetch main.star"): fetching signature: failed to download s3://fleet/main.star.sig: operation error S3: GetObject, https response error StatusCode: 404, RequestID: *, HostID: , NoSuchKey: * (glob)
```

## Clean up

```scrut
$ s3 rb --force s3://fleet >/dev/null && rm -rf "$store" ${store_root:+"$store_root"}
```
