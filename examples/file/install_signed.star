load("starcm", "file", "vars")

# The copy is checked in a private directory next to path, then moved into
# place, so what's installed is exactly what was checked.
r = file(
    label             = "install tool",
    path              = vars["managed_dir"] + "/bin/tool",
    source            = vars["source"],
    signer            = "macOS Software Signing",
    require_notarized = False,  # Apple doesn't notarize its own code
    mode              = 0o755,
    create_dirs       = True,
)
print("changed=%s signer=%s" % (r.changed, r.value.signer))
