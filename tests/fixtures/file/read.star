load("starcm", "file")

r = file(label = "read notes", path = "notes.txt", action = "read")
print("success=%s changed=%s" % (r.success, r.changed))
print(repr(r.value))
