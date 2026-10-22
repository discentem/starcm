load("starcm", "file", "here", "vars")

file(
    label  = "pinned",
    path   = vars["managed_dir"] + "/copy.txt",
    source = here("source.txt"),
    sha256 = "0000000000000000000000000000000000000000000000000000000000000000",
)
