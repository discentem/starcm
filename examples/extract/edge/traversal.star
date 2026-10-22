load("starcm", "extract", "here", "vars")

# evil.zip holds ok.txt and ../evil.txt.
extract(label = "unpack evil", archive = here("evil.zip"), dest = vars["managed_dir"] + "/out")
