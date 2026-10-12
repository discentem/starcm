load("starcm", "file")

def show(r):
    print("%s: success=%s changed=%s" % (r.label, r.success, r.changed))

show(file(label = "create", path = "out.txt", content = "v1\n"))
show(file(label = "same content", path = "out.txt", content = "v1\n"))
show(file(label = "new content", path = "out.txt", content = "v2\n"))
