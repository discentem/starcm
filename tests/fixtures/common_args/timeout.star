load("starcm", "exec")

exec(label = "short timeout", cmd = "true", args = [], timeout = "1ms")
r = exec(label = "no timeout", cmd = "sleep", args = ["0.2"])
print("no timeout: success=%s" % r.success)
