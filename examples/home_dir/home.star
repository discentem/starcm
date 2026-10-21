load("starcm", "file", "home_dir")

path = home_dir(".config", "agent.toml")
r = file(label = "agent config", path = path, content = "debug = false\n", create_dirs = True)
print("changed=%s" % r.changed)
