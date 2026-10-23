load("starcm", "install_remote_mac_pkg")

install_remote_mac_pkg(
    "s3://releases/agent-1.2.0.pkg",
    signature = "s3://elsewhere/agent-1.2.0.pkg.sig",
    signer    = "Developer ID Installer: Example Corp (ABCDE12345)",
    label     = "other bucket",
)
