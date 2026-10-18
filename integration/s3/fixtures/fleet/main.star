load("starcm", "exec", "load_dynamic", "s3_download", "vars")

# bootstrap.star fetches this file from s3://fleet and runs it. Everything
# it fetches is pinned by its signature, so a new release only needs new
# files and .sig files in the bucket, not a change here.

def fetch(key):
    """Fetches key and its signature, key.sig, into the local store."""
    return s3_download(
        label     = "fetch " + key,
        bucket    = "fleet",
        key       = key,
        signature = key + ".sig",
        endpoint  = vars["s3_endpoint"],
    ).value

web = load_dynamic(fetch("roles/web.star"), label = "load web role").value
web["configure"]()

exec(label = "install agent", cmd = "sh", args = [fetch("payloads/agent.sh")], live_output = True)
