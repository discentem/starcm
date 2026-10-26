load("starcm", "here", "path", "read_file")

for m in path.glob(here("third_party/*/go.mod")):
    print(read_file(m, label = "read go.mod").value.splitlines()[0])
