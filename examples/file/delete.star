load("starcm", "file", "vars")

for i in range(2):
    r = file(label = "remove agent settings", path = vars["managed_dir"] + "/agent.conf", action = "delete")
    print("pass %d: changed=%s message=%s" % (i + 1, r.changed, r.message))
