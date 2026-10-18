load("starcm", "s3_download")

r = s3_download(
    label    = "fetch agent from S3",
    bucket   = "docs",
    key      = "fixtures/download/files/agent-1.2.0.txt",
    sha256   = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
    save_to  = "agent-1.2.0.txt",
    # An S3-compatible store; leave this out for AWS S3.
    endpoint = "http://localhost:8000",
)
print(r.message)
