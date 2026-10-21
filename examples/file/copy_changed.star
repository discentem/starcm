load("starcm", "download", "exec", "file")

agent = download(
    label  = "fetch agent",
    url    = "http://localhost:8000/examples/download/files/agent-1.2.0.txt",
    sha256 = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
)

# Something changes the stored file after download() checked it...
exec(label = "tamper", cmd = "sh", args = ["-c", "chmod u+w \"$1\" && echo malware >> \"$1\"", "sh", agent.value])

# ...so file() refuses to copy it.
file(label = "install agent", path = "agent.txt", source = agent.value)
