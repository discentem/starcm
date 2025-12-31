e = ""

print('hello from b.star')

load("starcm", "load_dynamic")
load_dynamic("subfolder2/c.star", label="load c.star from b.star")