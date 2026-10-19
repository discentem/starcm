load("starcm", "write")

ADMINS = ["ada", "grace"]

def greet(name):
    write("hello, %s" % name, label = "greet " + name)
