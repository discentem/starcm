load("starcm", "check_mac_signature", "vars")

r = check_mac_signature(
    vars["path"],
    signer            = vars["signer"],
    require_notarized = vars.get("require_notarized", "yes") == "yes",
    label             = "check " + vars["path"],
)
print("%s (notarized=%s, kind=%s)" % (r.message, r.value.notarized, r.value.kind))
