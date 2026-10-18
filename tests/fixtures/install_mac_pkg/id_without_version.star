load("starcm", "install_mac_pkg")

install_mac_pkg(
    "agent.pkg",
    signer = "Developer ID Installer: Example Corp (ABCDE12345)",
    id     = "com.example.agent",
    label  = "no version",
)
