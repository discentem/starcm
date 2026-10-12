load("starcm", "file")

def show(r):
    print("%s: success=%s changed=%s" % (r.label, r.success, r.changed))

show(file(label = "create", path = "gone.txt", content = "bye\n"))
show(file(label = "delete", path = "gone.txt", action = "delete"))
show(file(label = "delete again", path = "gone.txt", action = "delete"))
