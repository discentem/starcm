load("starcm", "s3_download")

# The signature is another object in the same bucket.
r = s3_download(
    label     = "fetch signed agent from S3",
    bucket    = "docs",
    key       = "fixtures/download/files/agent-1.2.0.txt",
    signature = "fixtures/download/files/agent-1.2.0.txt.sig",
    save_to   = "signed/agent.txt",
    endpoint  = "http://localhost:8000",
)
print(r.message)
