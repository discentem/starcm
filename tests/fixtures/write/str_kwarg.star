load("starcm", "write")

r = write(str = "hello from str", label = "str kwarg")
print("success=%s changed=%s value=%s" % (r.success, r.changed, r.value))
