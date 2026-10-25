load("starcm", "glob", "here", "read_file")

for m in glob(here("third_party/*/go.mod"), label = "find modules").value:
    print(read_file(m, label = "read go.mod").value.splitlines()[0])
