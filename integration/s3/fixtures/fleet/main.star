load("starcm", "exec", "load_dynamic", "s3_download", "vars", "verify")

# bootstrap.star fetches this file from s3://fleet into fleet/ and runs it.
# Everything it fetches is pinned by sha256, and signed for --trusted-keys.

ENDPOINT = vars["s3_endpoint"]

def fetch_to_fleet(key, sha256):
    s3_download(
        label    = "fetch " + key,
        bucket   = "fleet",
        key      = key,
        sha256   = sha256,
        save_to  = "fleet/" + key,
        endpoint = ENDPOINT,
    )

# A role: the .star file and its signature, side by side, so --trusted-keys
# can check it when it's loaded.
fetch_to_fleet("roles/web.star", "551e439096d265d547833ef3918831680d0b88ab809c6e862aaa08e24aed815a")
fetch_to_fleet("roles/web.star.sig", "ff47f16399ac40cc19635676ee5e7d128fdf345e92caf9a104c7e3b15ebf180e")
web = load_dynamic("roles/web.star", label = "load web role").value
web["configure"]()

# A payload: into the local store, then checked against its signature
# before it runs.
agent = s3_download(
    label    = "fetch agent",
    bucket   = "fleet",
    key      = "payloads/agent.sh",
    sha256   = "cbdfc403eff5cab8029f427bca855bb33bdb998936bedf0c7d9c6150b36bd353",
    endpoint = ENDPOINT,
).value
fetch_to_fleet("payloads/agent.sh.sig", "07954fc37f06ddf3fd791ce9ecf1a9d4d3c0d35a8e824eea57db2e99d57552bf")
check = verify(agent, signature = "fleet/payloads/agent.sh.sig", label = "check agent")
if not check.success:
    fail(check.error)
exec(label = "install agent", cmd = "sh", args = [agent], live_output = True)
