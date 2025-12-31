load("starcm", "load_dynamic")
print('hello from a.star')

bstar = "subfolder/b.star"

load_dynamic(
    bstar, 
    label="load b.star",
    only_if = 1+2==2
)