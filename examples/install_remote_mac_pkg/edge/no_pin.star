load("starcm", "install_remote_mac_pkg")

install_remote_mac_pkg(
    "http://localhost:8000/examples/install_remote_mac_pkg/remote/unsigned.pkg",
    signer = "Developer ID Installer: Example Corp (ABCDE12345)",
    label  = "no pin",
)
