load("starcm", "download", "file")

agent = download(
    label  = "fetch agent",
    url    = "http://localhost:8000/examples/download/files/agent-1.2.0.txt",
    sha256 = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
)

# agent.value is a path in the store, so file() re-hashes it before copying.
r = file(
    label  = "install agent",
    path   = "agent.txt",
    source = agent.value,
    sha256 = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
)
print("changed=%s" % r.changed)
