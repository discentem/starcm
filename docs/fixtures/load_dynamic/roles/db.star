load("starcm", "write")

PORT = 5432

def configure():
    write("configuring database on port %d" % PORT, label = "db")
