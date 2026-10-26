load("starcm", "download", "sha256", "here", "vars")

# Hash a copy you trust once, then pin that hash in your configuration.
pinned = sha256(here("files/agent-1.2.0.json"))
print(pinned)

r = download(
    label   = "fetch manifest",
    url     = "http://localhost:8000/examples/download/files/agent-1.2.0.json",
    save_to = vars["managed_dir"] + "/manifest.json",
    sha256  = pinned,
)
print("success=%s changed=%s" % (r.success, r.changed))
