load("starcm", "load_dynamic")

for role in ["db", "web"]:
    mod = load_dynamic(module = "roles/%s.star" % role, label = "load " + role).value
    print("%s listens on %d" % (role, mod["PORT"]))
