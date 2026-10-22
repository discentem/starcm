load("starcm", "check_mac_signature", "here")

check_mac_signature(
    here("agent.dmg"),
    signer = "Developer ID Application: Example Corp (ABCDE12345)",
    kind   = "dmg",
    label  = "bad kind",
)
