load("starcm", "glob")

glob("third_party/*/go.mod", label = "relative")
