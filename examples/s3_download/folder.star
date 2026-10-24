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
