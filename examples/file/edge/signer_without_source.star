load("starcm", "file", "vars")

file(
    label   = "no source",
    path    = vars["managed_dir"] + "/tool",
    content = "x",
    signer  = "Developer ID Application: Example Corp (ABCDE12345)",
)
