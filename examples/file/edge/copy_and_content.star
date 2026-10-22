load("starcm", "file", "here", "vars")

file(label = "both", path = vars["managed_dir"] + "/copy.txt", source = here("source.txt"), content = "x")
