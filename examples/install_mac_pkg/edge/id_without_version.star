load("starcm", "here", "install_mac_pkg")

install_mac_pkg(
    here("agent.pkg"),
    signer = "Developer ID Installer: Example Corp (ABCDE12345)",
    id     = "com.example.agent",
    label  = "no version",
)
