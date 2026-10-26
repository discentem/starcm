load("starcm", "here", "path", "sha256")

for f in path.list(here("config"), pattern = "*.star"):
    h = sha256(here("config", f))
    print("%s  %s" % (h[:12], f))
