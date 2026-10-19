load("starcm", "download", "exec")

agent = download(
    label  = "fetch agent",
    url    = "http://localhost:8000/examples/download/files/agent-1.2.0.txt",
    sha256 = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
)

# agent.value is a path in the store, so exec() re-hashes it before running.
exec(label = "install agent", cmd = "cat", args = [agent.value], live_output = True)
