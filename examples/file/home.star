load("starcm", "file")

r = file(label = "dotfile", path = "~/.agentrc", content = "debug = false\n")
print("changed=%s" % r.changed)
