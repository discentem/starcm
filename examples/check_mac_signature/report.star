load("starcm", "check_mac_signature")

# /bin/ls is Apple's own code. Apple doesn't notarize what ships with
# macOS, so require_notarized must be off.
r = check_mac_signature(
    "/bin/ls",
    signer            = "macOS Software Signing",
    require_notarized = False,
    label             = "check ls",
)
print("%s, notarized=%s, %d-digit sha256" % (r.message, r.value.notarized, len(r.value.sha256)))
