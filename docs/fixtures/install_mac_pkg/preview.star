load("starcm", "install_mac_pkg", "vars")

r = install_mac_pkg(
    vars["pkg"],
    signer_sha256 = vars["signer"],
    what_if       = True,
    label         = "check " + vars["pkg"],
)
print(r.message)
print("team %s, notarized=%s" % (r.value.team_id, r.value.notarized))
