load("starcm", "download")

# No sha256 here: the signature, checked against --trusted-keys, says which
# sha256 the file must have.
r = download(
    label     = "fetch signed agent",
    url       = "http://localhost:8000/examples/download/files/agent-1.2.0.txt",
    signature = "http://localhost:8000/examples/download/files/agent-1.2.0.txt.sig",
    save_to   = "signed/agent.txt",
)
print(r.message)
