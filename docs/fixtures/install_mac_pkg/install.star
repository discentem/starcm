load("starcm", "download", "install_mac_pkg")

pkg = download(
    label  = "fetch agent",
    url    = "https://downloads.example.com/agent-1.2.0.pkg",
    sha256 = "ffcf2b6c3e96b1659e1bc5614ef23376f3d9b13fd65c3880b94a6318a9669e08",
)

r = install_mac_pkg(
    pkg.value,
    # The installer certificate's name, from `pkgutil --check-signature`.
    signer  = "Developer ID Installer: Example Corp (ABCDE12345)",
    id      = "com.example.agent",
    version = "1.2.0",
    label   = "install agent",
)
print("%s (changed=%s)" % (r.message, r.changed))
