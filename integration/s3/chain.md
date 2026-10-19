# s3_download(): setting up a laptop from a real S3 server

Runs a starcm deployment end to end against a real S3-compatible server,
[rustfs](https://github.com/rustfs/rustfs), which checks every request's
signature. This runs only in the scrut CI workflow
(`.github/workflows/scrut.yml`). That workflow starts rustfs, then sets
`STARCM_TEST_S3_ENDPOINT` and test credentials in `AWS_ACCESS_KEY_ID` and
`AWS_SECRET_ACCESS_KEY`. It isn't part of the regular `tests/` run.

It plays a laptop being set up from configuration in an S3 bucket. The
laptop starts with only `bootstrap.star`. That file fetches `main.star` from
the `config` bucket into the local store and runs it. `main.star` fetches two
more files the same way:

- `lib/settings.star`, a module that writes the agent's settings file;
- `scripts/install-agent.sh`, a script it runs to install the agent.

Everything runs under `--trusted-keys`. No file is pinned by sha256: each is
pinned by its signature, `<key>.sig` in the same bucket, so a new release
only needs new files and signatures in the bucket.

## Set up the bucket

Act as the admin publishing the configuration: sign each file with
`starcm sign`, which writes a `.sig` next to it, then upload the files and
signatures to the bucket. The key is the test-only one in
`tests/fixtures/signing/`; the laptop trusts its public half
(`--trusted-keys trusted`). The local copy is then removed, so everything has
to come from the bucket.

```scrut
$ cp -R "$TESTDIR/fixtures/." "$TMPDIR" && cd "$TMPDIR" && mkdir trusted && cp "$TESTDIR/../../tests/fixtures/signing/trusted/test.pub.pem" trusted/ && key="$TESTDIR/../../tests/fixtures/signing/private/test.pem" && untrusted="$TESTDIR/../../tests/fixtures/signing/private/untrusted.pem"
```

```scrut
$ (cd config && starcm sign --expires 30d --key "$key" main.star lib/settings.star scripts/install-agent.sh) && starcm sign --expires 30d --key "$key" bootstrap.star
signed main.star with key "test" until * -> main.star.sig (glob)
signed lib/settings.star with key "test" until * -> lib/settings.star.sig (glob)
signed scripts/install-agent.sh with key "test" until * -> scripts/install-agent.sh.sig (glob)
signed bootstrap.star with key "test" until * -> bootstrap.star.sig (glob)
```

```scrut
$ s3() { aws --endpoint-url "$STARCM_TEST_S3_ENDPOINT" s3 "$@"; } && s3 mb s3://config >/dev/null && s3 sync --quiet config s3://config && rm -rf config && s3 ls --recursive s3://config | awk '{print $4}' | sort
lib/settings.star
lib/settings.star.sig
main.star
main.star.sig
scripts/install-agent.sh
scripts/install-agent.sh.sig
```

Fetched files go into a local store. On the Windows CI runners, starcm
refuses a store under `D:\` (where `$TMPDIR` is) because another account owns
the drive, so there the store goes under `%LocalAppData%`.

```scrut
$ store="$TMPDIR/store"; case "$(uname -s)" in MINGW*|MSYS*|CYGWIN*) store_root=$(mktemp -d "$(cygpath -u "$LOCALAPPDATA")/starcm-s3.XXXXXX") && store="$store_root/store" ;; esac; run() { starcm --trusted-keys trusted --store-dir "$store" --var "s3_endpoint=$STARCM_TEST_S3_ENDPOINT" bootstrap.star; }
```

## The laptop gets set up

```scrut
$ run
agent settings: changed=True
agent 1.0 installed
```

The only thing created where it ran is the settings file (the store sits
there too on macOS and Linux, so it's left out of the listing):

```scrut
$ ls | grep -vx store && cat managed/agent.conf
bootstrap.star
bootstrap.star.sig
managed
trusted
server = https://config.example.com
channel = stable
```

Each fetched file is in the store, with its signature next to it, which is
how `--trusted-keys` checked `main.star` and `settings.star` before loading
them:

```scrut
$ find "$store" -type f | sed 's|.*[/\\]||' | sort
install-agent.sh
install-agent.sh.sig
main.star
main.star.sig
settings.star
settings.star.sig
```

A second run finds every file in the store with its signed sha256, so it
only fetches the signatures again, and the settings file is already right:

```scrut
$ run
agent settings: changed=False
agent 1.0 installed
```

## A new release needs no configuration change

Publish agent 2.0: upload the new install script and its new signature.
Nothing on the laptop changes.

```scrut
$ printf '#!/bin/sh\necho "agent 2.0 installed"\n' > install-agent.sh && starcm sign --expires 30d --key "$key" install-agent.sh >/dev/null && s3 cp --quiet install-agent.sh s3://config/scripts/install-agent.sh && s3 cp --quiet install-agent.sh.sig s3://config/scripts/install-agent.sh.sig && run
agent settings: changed=False
agent 2.0 installed
```

## A tampered bucket is refused

Someone with write access to the bucket swaps in an install script that the
signature doesn't cover. While the signed release is already in the store,
that one runs and the swap changes nothing:

```scrut
$ printf '#!/bin/sh\necho pwned\n' > evil.sh && s3 cp --quiet evil.sh s3://config/scripts/install-agent.sh && run
agent settings: changed=False
agent 2.0 installed
```

On a laptop that doesn't have it yet, the download doesn't match the sha256
the signature records, so it's refused before it's stored or run:

```scrut
$ rm -rf "$store" && run 2>&1 | tail -1
Error in s3_download: expected sha256 hash *, got * (glob)
```

Signing the swap with a key the machine doesn't trust doesn't help:

```scrut
$ starcm sign --expires 30d --key "$untrusted" evil.sh >/dev/null && s3 cp --quiet evil.sh.sig s3://config/scripts/install-agent.sh.sig && run 2>&1 | tail -1
Error in s3_download: s3_download(label="fetch scripts/install-agent.sh"): signature scripts/install-agent.sh.sig: key "untrusted" is not trusted
```

## The server checks credentials

```scrut
$ AWS_SECRET_ACCESS_KEY=wrong run 2>&1 | tail -1
Error in s3_download: s3_download(label="fetch main.star"): fetching signature: failed to download s3://config/main.star.sig: operation error S3: GetObject, https response error StatusCode: 403, RequestID: *, HostID: , api error SignatureDoesNotMatch: * (glob)
```

```scrut
$ AWS_ACCESS_KEY_ID=nobody run 2>&1 | tail -1
Error in s3_download: s3_download(label="fetch main.star"): fetching signature: failed to download s3://config/main.star.sig: operation error S3: GetObject, https response error StatusCode: 403, RequestID: *, HostID: , api error InvalidAccessKeyId: * (glob)
```

## A missing object fails the run

```scrut
$ s3 rm --quiet s3://config/main.star.sig && run 2>&1 | tail -1
Error in s3_download: s3_download(label="fetch main.star"): fetching signature: failed to download s3://config/main.star.sig: operation error S3: GetObject, https response error StatusCode: 404, RequestID: *, HostID: , NoSuchKey: * (glob)
```

## Clean up

```scrut
$ s3 rb --force s3://config >/dev/null && rm -rf "$store" ${store_root:+"$store_root"}
```
