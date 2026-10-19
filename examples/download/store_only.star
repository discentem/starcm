load("starcm", "download")

download(
    label   = "fetch agent to a path",
    url     = "http://localhost:8000/examples/download/files/agent-1.2.0.txt",
    save_to = "agent-1.2.0.txt",
    sha256  = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
)
