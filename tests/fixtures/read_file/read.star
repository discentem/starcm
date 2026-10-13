load("starcm", "read_file")

r = read_file("notes.txt", label = "read notes")
print("success=%s changed=%s" % (r.success, r.changed))
print(repr(r.value))
print(len(read_file(path = "notes.txt", label = "path= kwarg").value.splitlines()))
