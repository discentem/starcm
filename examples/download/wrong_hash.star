load("starcm", "download")

download(
    label   = "fetch agent",
    url     = "http://localhost:8000/examples/download/files/agent-1.2.0.txt",
    save_to = "tampered.txt",
    sha256  = "0000000000000000000000000000000000000000000000000000000000000000",
)
print("never reached")
