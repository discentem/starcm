# Running a chain of `.star` files from an S3 bucket

This md file is actually a [scrut](https://github.com/facebookincubator/scrut)
test. The code blocks are executed by `scrut` in CI to test the following
things.

It retrieves a chain of `.star` files from an emulated S3 bucket and runs the
workflow. The bucket is served by [rustfs](https://github.com/rustfs/rustfs),
an S3-compatible server that checks credentials on every request, like real
S3. CI (`.github/workflows/scrut.yml`) starts rustfs on `127.0.0.1:9000` and sets
test credentials before running this file.

The chain:

```text
bootstrap.star                 local; starcm runs this. It fetches main.star
                               from the bucket and runs it.
s3://config/
  main.star                    fetches the two files below and uses them
  lib/settings.star            writes the agent's settings file
  scripts/install-agent.sh     installs the agent
```

## The files

`bootstrap.star`, the file starcm runs:

<!-- fixture: fixtures/bootstrap.star -->
```python
load("starcm", "load_dynamic", "s3_download", "vars")

# The only configuration on the machine to start with. main.star comes from
# s3://config into the local store, pinned by its signature, and runs from
# there: its .sig is stored next to it for --trusted-keys.
main = s3_download(
    label     = "fetch main.star",
    bucket    = "config",
    key       = "main.star",
    signature = "main.star.sig",
    endpoint  = vars["s3_endpoint"],
).value

load_dynamic(main, label = "run main.star")
```

`main.star`, in the bucket:

<!-- fixture: fixtures/config/main.star -->
```python
load("starcm", "exec", "load_dynamic", "s3_download", "vars")

# bootstrap.star fetches this file from s3://config and runs it. Everything
# it fetches is pinned by its signature, so a new release only needs new
# files and .sig files in the bucket, not a change here.

def fetch(key):
    """Fetches key and its signature, key.sig, into the local store."""
    return s3_download(
        label     = "fetch " + key,
        bucket    = "config",
        key       = key,
        signature = key + ".sig",
        endpoint  = vars["s3_endpoint"],
    ).value

settings = load_dynamic(fetch("lib/settings.star"), label = "load settings").value
settings["apply"]()

exec(label = "install agent", cmd = "sh", args = [fetch("scripts/install-agent.sh")], live_output = True)
```

`lib/settings.star`, in the bucket:

<!-- fixture: fixtures/config/lib/settings.star -->
```python
load("starcm", "dedent", "file")

def apply():
    """Writes the agent's settings file and reports whether it changed."""
    r = file(
        label       = "agent settings",
        path        = "managed/agent.conf",
        content     = dedent("""
            server = https://config.example.com
            channel = stable
            """),
        create_dirs = True,
    )
    print("agent settings: changed=%s" % r.changed)
```

`scripts/install-agent.sh`, in the bucket:

<!-- fixture: fixtures/config/scripts/install-agent.sh -->
```sh
#!/bin/sh
# Stands in for a real installer: it only prints which version it installed.
echo "agent 1.0 installed"
```

starcm runs with `--trusted-keys`, so every `.star` file must be signed by a
trusted key. Each download is pinned by its signature (`<file>.sig`, next to
it in the bucket) rather than by a sha256.

## Set up

Copy the files above into scrut's scratch directory and work there:

```scrut
$ cp -R "$TESTDIR/fixtures/." "$TMPDIR" && cd "$TMPDIR"
```

Signing uses the test-only keys from `tests/fixtures/signing/`:
`keys/test.pem` signs the configuration, `trusted/` holds its public half for
`--trusted-keys`, and `keys/untrusted.pem` is a key starcm doesn't trust.

```scrut
$ mkdir keys trusted && cp "$TESTDIR"/../../tests/fixtures/signing/private/test.pem "$TESTDIR"/../../tests/fixtures/signing/private/untrusted.pem keys/ && cp "$TESTDIR"/../../tests/fixtures/signing/trusted/test.pub.pem trusted/
```

## Publish the configuration to the bucket

```scrut
$ aws --endpoint-url http://127.0.0.1:9000 s3 mb s3://config
make_bucket: config
```

Sign each file in `config/`, which writes a `.sig` next to it:

```scrut
$ (cd config && starcm sign --expires 30d --key ../keys/test.pem main.star lib/settings.star scripts/install-agent.sh)
signed main.star with key "test" until * -> main.star.sig (glob)
signed lib/settings.star with key "test" until * -> lib/settings.star.sig (glob)
signed scripts/install-agent.sh with key "test" until * -> scripts/install-agent.sh.sig (glob)
```

Upload the folder, files and signatures, to the bucket:

```scrut
$ aws --endpoint-url http://127.0.0.1:9000 s3 sync --quiet config s3://config && aws --endpoint-url http://127.0.0.1:9000 s3 ls --recursive s3://config | awk '{print $4}' | sort
lib/settings.star
lib/settings.star.sig
main.star
main.star.sig
scripts/install-agent.sh
scripts/install-agent.sh.sig
```

`bootstrap.star` isn't downloaded, it's already here. But because starcm runs
with `--trusted-keys`, it has to be signed like every other `.star` file:

```scrut
$ starcm sign --expires 30d --key keys/test.pem bootstrap.star
signed bootstrap.star with key "test" until * -> bootstrap.star.sig (glob)
```

## Run the chain

```scrut
$ starcm --trusted-keys trusted --store-dir store --var s3_endpoint=http://127.0.0.1:9000 bootstrap.star
agent settings: changed=True
agent 1.0 installed
```

## See what happened

`lib/settings.star` wrote the agent's settings file:

```scrut
$ cat managed/agent.conf
server = https://config.example.com
channel = stable
```

Each downloaded file is in starcm's local store, `store/`, in a folder named
after its sha256, with its signature next to it:

```scrut
$ ls store/sha256/*/
store/sha256/ac2c374ce948e9994baff90748c8fb577ee7244bff12c287d782686fa5947f5b/:
main.star
main.star.sig

store/sha256/b4ef42f0602b30edd49326051dbdab600937b8bd7a3273353a748efd3b3ad96b/:
settings.star
settings.star.sig

store/sha256/fde98a8ca07d684f266eb29ca7a62e4b7e28dcd5f5b130916e343837b311af4a/:
install-agent.sh
install-agent.sh.sig
```

## Running it again changes nothing

Every file is already in the store, so starcm only fetches the signatures
again:

```scrut
$ starcm --trusted-keys trusted --store-dir store --var s3_endpoint=http://127.0.0.1:9000 bootstrap.star
agent settings: changed=False
agent 1.0 installed
```

## A new release only needs a new upload

Publish agent 2.0. Nothing in `bootstrap.star` or `main.star` changes:

```scrut
$ printf 'echo "agent 2.0 installed"\n' > config/scripts/install-agent.sh && starcm sign --expires 30d --key keys/test.pem config/scripts/install-agent.sh
signed config/scripts/install-agent.sh with key "test" until * -> config/scripts/install-agent.sh.sig (glob)
```

```scrut
$ aws --endpoint-url http://127.0.0.1:9000 s3 cp --quiet config/scripts/install-agent.sh s3://config/scripts/install-agent.sh && aws --endpoint-url http://127.0.0.1:9000 s3 cp --quiet config/scripts/install-agent.sh.sig s3://config/scripts/install-agent.sh.sig
```

```scrut
$ starcm --trusted-keys trusted --store-dir store --var s3_endpoint=http://127.0.0.1:9000 bootstrap.star
agent settings: changed=False
agent 2.0 installed
```

## Tampering is refused

Someone with access to the bucket replaces the install script, without a
valid signature:

```scrut
$ printf 'echo pwned\n' > evil.sh && aws --endpoint-url http://127.0.0.1:9000 s3 cp --quiet evil.sh s3://config/scripts/install-agent.sh
```

Agent 2.0 is already in the store, so starcm keeps using it:

```scrut
$ starcm --trusted-keys trusted --store-dir store --var s3_endpoint=http://127.0.0.1:9000 bootstrap.star
agent settings: changed=False
agent 2.0 installed
```

With an empty store, starcm downloads the replaced script and refuses it,
because it doesn't match the sha256 its signature records:

```scrut
$ rm -rf store && starcm --trusted-keys trusted --store-dir store --var s3_endpoint=http://127.0.0.1:9000 bootstrap.star 2>&1 | tail -1
Error in s3_download: expected sha256 hash *, got * (glob)
```

Signing the replacement with a key starcm doesn't trust doesn't help:

```scrut
$ starcm sign --expires 30d --key keys/untrusted.pem evil.sh && aws --endpoint-url http://127.0.0.1:9000 s3 cp --quiet evil.sh.sig s3://config/scripts/install-agent.sh.sig
signed evil.sh with key "untrusted" until * -> evil.sh.sig (glob)
```

```scrut
$ starcm --trusted-keys trusted --store-dir store --var s3_endpoint=http://127.0.0.1:9000 bootstrap.star 2>&1 | tail -1
Error in s3_download: s3_download(label="fetch scripts/install-agent.sh"): signature scripts/install-agent.sh.sig: key "untrusted" is not trusted
```

## The server checks credentials

A wrong secret key:

```scrut
$ AWS_SECRET_ACCESS_KEY=wrong starcm --trusted-keys trusted --store-dir store --var s3_endpoint=http://127.0.0.1:9000 bootstrap.star 2>&1 | grep -o 'api error [A-Za-z]*'
api error SignatureDoesNotMatch
```

A key the server doesn't know:

```scrut
$ AWS_ACCESS_KEY_ID=nobody starcm --trusted-keys trusted --store-dir store --var s3_endpoint=http://127.0.0.1:9000 bootstrap.star 2>&1 | grep -o 'api error [A-Za-z]*'
api error InvalidAccessKeyId
```

## A missing file fails the run

```scrut
$ aws --endpoint-url http://127.0.0.1:9000 s3 rm --quiet s3://config/main.star.sig && starcm --trusted-keys trusted --store-dir store --var s3_endpoint=http://127.0.0.1:9000 bootstrap.star 2>&1 | grep -o 'NoSuchKey'
NoSuchKey
```

## Clean up

```scrut
$ aws --endpoint-url http://127.0.0.1:9000 s3 rb --force s3://config >/dev/null
```
