load("starcm", "check_mac_signature")

check_mac_signature(
    "agent.dmg",
    signer = "Developer ID Application: Example Corp (ABCDE12345)",
    kind   = "dmg",
    label  = "bad kind",
)
