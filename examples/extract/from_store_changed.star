load("starcm", "download", "exec", "extract", "vars")

tool = download(
    label  = "fetch tool",
    url    = "http://localhost:8000/examples/extract/files/tool-1.0.zip",
    sha256 = "da3bf2e6a81a33c97858840bdfc119be2c127638e62595e18130af760d51a7f9",
)

# Something changes the stored archive after download() checked it...
exec(label = "tamper", cmd = "sh", args = ["-c", "chmod u+w \"$1\" && echo malware >> \"$1\"", "sh", tool.value])

# ...so extract() refuses to read it.
extract(label = "unpack tool", archive = tool.value, dest = vars["managed_dir"] + "/tampered-out")
