load("starcm", "file")

file(
    label  = "pinned",
    path   = "copy.txt",
    source = "source.txt",
    sha256 = "0000000000000000000000000000000000000000000000000000000000000000",
)
