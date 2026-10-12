load("starcm", "file")

r = file(label = "nested", path = "a/b/c.txt", content = "deep\n", create_dirs = True)
print("success=%s changed=%s" % (r.success, r.changed))
