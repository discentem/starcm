load("starcm", "here", "install_mac_pkg")

install_mac_pkg(
    here("unsigned.pkg"),
    signer = "Developer ID Installer: Example Corp (ABCDE12345)",
    label  = "unsigned",
)
print("never reached")
