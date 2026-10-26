load("starcm", "check_mac_signature", "vars")

sig = check_mac_signature(
    vars["path"],
    signer            = vars["signer"],
    require_notarized = vars.get("require_notarized", "yes") == "yes",
)
print("signed by %r (notarized=%s, kind=%s)" % (sig.signer, sig.notarized, sig.kind))
