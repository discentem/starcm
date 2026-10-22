load("starcm", "read_file", "here")

r = read_file(here("notes.txt"), label = "read notes")
print("success=%s changed=%s" % (r.success, r.changed))
print(repr(r.value))
print(r.value.splitlines())
