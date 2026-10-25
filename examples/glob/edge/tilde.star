load("starcm", "glob")

for m in glob("~/third_party/*/go.mod", label = "under home").value:
    print(m)
