load("starcm", "exec")

r = exec(label = "print the date", cmd = "echo", args = ["2026-10-17"])
print(r)
print("%s: success=%s changed=%s" % (r.label, r.success, r.changed))
