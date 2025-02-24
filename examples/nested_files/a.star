load("starcm", "load_dynamic")
print('hello from a.star')

bstar = "subfolder/b.star"

load_dynamic(path=bstar, name="load %s" % bstar)