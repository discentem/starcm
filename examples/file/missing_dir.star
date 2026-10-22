load("starcm", "file", "vars")

file(label = "no parent", path = vars["managed_dir"] + "/nowhere/file.txt", content = "x")
