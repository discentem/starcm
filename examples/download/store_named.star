load("starcm", "download")

r = download(
    label     = "fetch agent under another name",
    url       = "http://localhost:8000/examples/download/files/agent-1.2.0.txt",
    sha256    = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
    file_name = "agent.txt",
)
print(r.value.replace("\\", "/"))
