load("starcm", "download", "sha256")

# Hash a copy you trust once, then pin that hash in your configuration.
pinned = sha256("files/agent-1.2.0.json", label = "hash trusted copy").value
print(pinned)

r = download(
    label   = "fetch manifest",
    url     = "http://localhost:8000/docs/fixtures/download/files/agent-1.2.0.json",
    save_to = "manifest.json",
    sha256  = pinned,
)
print("success=%s changed=%s" % (r.success, r.changed))
