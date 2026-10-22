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
