load("starcm", "download", "vars")

VERSION = "1.2.0"
SHA256  = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08"

r = download(
    label   = "fetch agent %s" % VERSION,
    url     = "%s/agent-%s.txt" % (vars["base_url"], VERSION),
    save_to = vars["managed_dir"] + "/agent-%s.txt" % VERSION,
    sha256  = SHA256,
)
print("%s: success=%s changed=%s" % (r.label, r.success, r.changed))
