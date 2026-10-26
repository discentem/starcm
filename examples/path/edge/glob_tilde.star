load("starcm", "path")

for m in path.glob("~/glob/third_party/*/go.mod"):
    print(m)
