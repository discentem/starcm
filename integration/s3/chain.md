# s3_download(): a signed fleet served from a real S3 server

Runs a starcm deployment end to end against a real S3-compatible server,
[rustfs](https://github.com/rustfs/rustfs), which checks every request's
signature. This runs only in the scrut CI workflow
(`.github/workflows/scrut.yml`). That workflow starts rustfs, then sets
`STARCM_TEST_S3_ENDPOINT` and test credentials in `AWS_ACCESS_KEY_ID` and
`AWS_SECRET_ACCESS_KEY`. It isn't part of the regular `tests/` run.

The machine starts with only `bootstrap.star`. That file fetches `main.star`
and its signature from the `fleet` bucket, then runs it. `main.star` fetches a
role and a payload the same way, checks the payload's signature, and runs it.
Everything runs under `--trusted-keys`, and every file is pinned by sha256.

## Set up the bucket

Sign the fleet with the test key from `tests/fixtures/signing/`. A fixed
expiry keeps the signatures, and so the pinned sha256s of the `.sig` files,
identical on every run. Then upload the fleet, and remove the local copy, so
everything has to come from the bucket.

```scrut
$ cp -R "$TESTDIR/fixtures/." "$TMPDIR" && cd "$TMPDIR" && mkdir trusted && cp "$TESTDIR/../../tests/fixtures/signing/trusted/test.pub.pem" trusted/ && key="$TESTDIR/../../tests/fixtures/signing/private/test.pem"
```

```scrut
$ (cd fleet && starcm sign --expires 2099-01-01 --key "$key" main.star roles/web.star payloads/agent.sh) && starcm sign --expires 2099-01-01 --key "$key" bootstrap.star
signed main.star with key "test" until 2099-01-01T00:00:00Z -> main.star.sig
signed roles/web.star with key "test" until 2099-01-01T00:00:00Z -> roles/web.star.sig
signed payloads/agent.sh with key "test" until 2099-01-01T00:00:00Z -> payloads/agent.sh.sig
signed bootstrap.star with key "test" until 2099-01-01T00:00:00Z -> bootstrap.star.sig
```

```scrut
$ s3() { aws --endpoint-url "$STARCM_TEST_S3_ENDPOINT" "$@"; } && s3 s3 mb s3://fleet >/dev/null && s3 s3 sync --quiet fleet s3://fleet && rm -rf fleet && s3 s3 ls --recursive s3://fleet | awk '{print $4}' | sort
main.star
main.star.sig
payloads/agent.sh
payloads/agent.sh.sig
roles/web.star
roles/web.star.sig
```

Stored payloads go into a local store. On the Windows CI runners, starcm
refuses a store under `D:\` (where `$TMPDIR` is) because another account owns
the drive, so there the store goes under `%LocalAppData%`.

```scrut
$ store="$TMPDIR/store"; case "$(uname -s)" in MINGW*|MSYS*|CYGWIN*) store_root=$(mktemp -d "$(cygpath -u "$LOCALAPPDATA")/starcm-s3.XXXXXX") && store="$store_root/store" ;; esac
```

## The fleet runs

```scrut
$ starcm --trusted-keys trusted --store-dir "$store" --var "s3_endpoint=$STARCM_TEST_S3_ENDPOINT" bootstrap.star
web role configured
agent 1.0 installed
```

Everything it fetched is in place:

```scrut
$ find fleet -type f | sort
fleet/main.star
fleet/main.star.sig
fleet/payloads/agent.sh.sig
fleet/roles/web.star
fleet/roles/web.star.sig
```

A second run finds every file with the right hash and fetches nothing, but
still runs the role and the payload:

```scrut
$ starcm --trusted-keys trusted --store-dir "$store" --var "s3_endpoint=$STARCM_TEST_S3_ENDPOINT" bootstrap.star
web role configured
agent 1.0 installed
```

## The server checks credentials

A wrong secret is refused by the server:

```scrut
$ rm -rf fleet && AWS_SECRET_ACCESS_KEY=wrong starcm --trusted-keys trusted --store-dir "$store" --var "s3_endpoint=$STARCM_TEST_S3_ENDPOINT" bootstrap.star 2>&1 | tail -1
Error in s3_download: failed to download s3://fleet/main.star: operation error S3: GetObject, https response error StatusCode: 403, RequestID: *, HostID: , api error SignatureDoesNotMatch: * (glob)
```

So is a key the server doesn't know:

```scrut
$ AWS_ACCESS_KEY_ID=nobody starcm --trusted-keys trusted --store-dir "$store" --var "s3_endpoint=$STARCM_TEST_S3_ENDPOINT" bootstrap.star 2>&1 | tail -1
Error in s3_download: failed to download s3://fleet/main.star: operation error S3: GetObject, https response error StatusCode: 403, RequestID: *, HostID: , api error InvalidAccessKeyId: * (glob)
```

## A tampered bucket is refused

Someone with write access to the bucket swaps in a new payload. The sha256
pinned in `main.star` stops it before it's stored or run, after the role
has already run. The store is cleared first, since a payload that's already
stored isn't fetched again.

```scrut
$ rm -rf "$store" && printf '#!/bin/sh\necho pwned\n' > evil.sh && aws --endpoint-url "$STARCM_TEST_S3_ENDPOINT" s3 cp --quiet evil.sh s3://fleet/payloads/agent.sh && starcm --trusted-keys trusted --store-dir "$store" --var "s3_endpoint=$STARCM_TEST_S3_ENDPOINT" bootstrap.star 2>&1 | grep -v '^  '
web role configured
Traceback (most recent call last):
Error in load_dynamic: load failed:
Traceback (most recent call last):
Error in s3_download: expected sha256 hash cbdfc403eff5cab8029f427bca855bb33bdb998936bedf0c7d9c6150b36bd353, got * (glob)
```

The same goes for `main.star` itself, which `bootstrap.star` pins:

```scrut
$ rm -rf fleet && echo '# edited' > main.star && aws --endpoint-url "$STARCM_TEST_S3_ENDPOINT" s3 cp --quiet main.star s3://fleet/main.star && starcm --trusted-keys trusted --store-dir "$store" --var "s3_endpoint=$STARCM_TEST_S3_ENDPOINT" bootstrap.star 2>&1 | tail -1
Error in s3_download: expected sha256 hash c0f6f51a8336ea11bd03840f9399f71c65f0c000338160968537109b76ffb641, got * (glob)
```

## A missing object fails the run

```scrut
$ aws --endpoint-url "$STARCM_TEST_S3_ENDPOINT" s3 rm --quiet s3://fleet/main.star && starcm --trusted-keys trusted --store-dir "$store" --var "s3_endpoint=$STARCM_TEST_S3_ENDPOINT" bootstrap.star 2>&1 | tail -1
Error in s3_download: failed to download s3://fleet/main.star: operation error S3: GetObject, https response error StatusCode: 404, RequestID: *, HostID: , NoSuchKey: * (glob)
```

## Clean up

```scrut
$ aws --endpoint-url "$STARCM_TEST_S3_ENDPOINT" s3 rb --force s3://fleet >/dev/null && rm -rf "$store" ${store_root:+"$store_root"}
```
