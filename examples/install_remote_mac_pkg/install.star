load("starcm", "install_remote_mac_pkg")

r = install_remote_mac_pkg(
    "https://releases.example.com/agent-1.2.0.pkg",
    # No sha256 here: the signature, checked against --trusted-keys, says
    # which sha256 the package must have.
    signature = "https://releases.example.com/agent-1.2.0.pkg.sig",
    # The installer certificate's name, from `pkgutil --check-signature`.
    signer    = "Developer ID Installer: Example Corp (ABCDE12345)",
    id        = "com.example.agent",
    version   = "1.2.0",
    label     = "install agent",
)
print("%s (changed=%s)" % (r.message, r.changed))
