load("starcm", "read_file")

r = read_file("notes.txt", label = "read notes")
print("success=%s changed=%s" % (r.success, r.changed))
print(repr(r.value))
print(r.value.splitlines())
