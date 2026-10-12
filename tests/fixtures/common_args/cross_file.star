load("starcm", "exec")
load("lib.star", "call_with_timeout")

call_with_timeout()
r = exec(label = "main call, no timeout", cmd = "sleep", args = ["1"])
print("main call: success=%s" % r.success)
