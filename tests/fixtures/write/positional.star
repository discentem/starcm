load("starcm", "write")

r = write("hello from write", label = "positional")
print("success=%s changed=%s value=%s" % (r.success, r.changed, r.value))
