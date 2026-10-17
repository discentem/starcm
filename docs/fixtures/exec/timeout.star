load("starcm", "exec")

r = exec(label = "hangs", cmd = "sleep", args = ["2"], timeout = "500ms")
print("success=%s error=%s" % (r.success, r.error))
