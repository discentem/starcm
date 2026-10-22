load("starcm", "list_files", "sha256", "here")

for f in list_files(here("config"), pattern = "*.star", label = "find config").value:
    h = sha256(here("config", f), label = "hash " + f).value
    print("%s  %s" % (h[:12], f))
