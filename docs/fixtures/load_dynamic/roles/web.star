load("starcm", "write")

PORT = 443

def configure():
    write("configuring web server on port %d" % PORT, label = "web")
