load("starcm", "download", "extract", "vars")

tool = download(
    label  = "fetch tool",
    url    = "http://localhost:8000/examples/extract/files/tool-1.0.zip",
    sha256 = "da3bf2e6a81a33c97858840bdfc119be2c127638e62595e18130af760d51a7f9",
)

# tool.value is a path in the store, so extract() re-hashes it before reading.
r = extract(label = "unpack tool", archive = tool.value, dest = vars["managed_dir"] + "/from-store")
print("changed=%s files=%d" % (r.changed, len(r.value)))
