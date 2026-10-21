load("starcm", "extract")

# evil.zip holds ok.txt and ../evil.txt.
extract(label = "unpack evil", archive = "evil.zip", dest = "out")
