load("starcm", "install_mac_pkg")

# A Team ID is not enough: the installer certificate's sha256 is required.
install_mac_pkg("agent.pkg", signer_sha256 = "W5364U7YZB", label = "team id")
