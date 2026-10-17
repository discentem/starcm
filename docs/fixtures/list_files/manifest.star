load("starcm", "list_files", "sha256")

for f in list_files("config", pattern = "*.star", label = "find config").value:
    h = sha256("config/" + f, label = "hash " + f).value
    print("%s  %s" % (h[:12], f))
