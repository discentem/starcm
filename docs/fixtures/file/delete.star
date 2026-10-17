load("starcm", "file")

for i in range(2):
    r = file(label = "remove old config", path = "hosts.allow", action = "delete")
    print("pass %d: changed=%s message=%s" % (i + 1, r.changed, r.message))
