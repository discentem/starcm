load("starcm", "exec")

r = exec(label = "echo", cmd = "sh", args = ["-c", "echo captured"])
print("success=%s changed=%s value=%r" % (r.success, r.changed, r.value))
