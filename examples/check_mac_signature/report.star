load("starcm", "check_mac_signature")

# /bin/ls is Apple's own code. Apple doesn't notarize what ships with
# macOS, so require_notarized must be off.
sig = check_mac_signature(
    "/bin/ls",
    signer            = "macOS Software Signing",
    require_notarized = False,
)
print("signed by %r, notarized=%s, %d-digit sha256" % (sig.signer, sig.notarized, len(sig.sha256)))
