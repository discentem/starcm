load("starcm", "install_mac_pkg")

install_mac_pkg(
    "agent.pkg",
    signer_sha256 = "1E 77 DE 6F E4 7D 42 69 84 C5 AA 18 0D 58 1F 32 46 19 7F 73 C0 C7 13 27 7B 84 40 08 B9 36 E0 93",
    id            = "com.example.agent",
    label         = "no version",
)
