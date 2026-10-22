load("starcm", "exec", "load_remote", "s3_download", "vars")

# bootstrap.star fetches this file from s3://config and runs it. Everything
# it fetches is pinned by its signature, so a new release only needs new
# files and .sig files in the bucket, not a change here.

def fetch(key):
    """Fetches key and its signature, key.sig, into the local store and
    returns its path there."""
    return s3_download(
        label     = "fetch " + key,
        bucket    = "config",
        key       = key,
        signature = key + ".sig",
        endpoint  = vars["s3_endpoint"],
    ).value

settings = load_remote(
    label     = "load settings",
    url       = "s3://config/lib/settings.star",
    signature = "s3://config/lib/settings.star.sig",
    endpoint  = vars["s3_endpoint"],
).value
settings["apply"]()

exec(label = "install agent", cmd = "sh", args = [fetch("scripts/install-agent.sh")], live_output = True)
