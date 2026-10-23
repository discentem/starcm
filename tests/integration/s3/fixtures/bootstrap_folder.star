load("starcm", "load_remote", "s3_download", "vars")

# One signature, on the folder's manifest, pins every file under
# s3://config/agent/. load_remote runs main.star from the folder in the store.
agent = load_remote(
    label     = "run the agent folder",
    url       = "s3://config/agent/",
    entry     = "main.star",
    signature = "s3://config/agent/starcm-manifest.json.sig",
    endpoint  = vars["s3_endpoint"],
).value
agent["apply"]()

# s3_download fetches the same folder to a directory of your choosing.
r = s3_download(
    label     = "copy the agent folder",
    bucket    = "config",
    key       = "agent/",
    signature = "agent/starcm-manifest.json.sig",
    save_to   = vars["managed_dir"] + "/agent",
    endpoint  = vars["s3_endpoint"],
)
print("agent copy: changed=%s" % r.changed)
