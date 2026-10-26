load("starcm", "here", "path")

print(path.glob(here("../glob/third_party/*/Cargo.toml")))   # nothing matches
print(path.glob(here("../glob/missing/*/go.mod")))           # missing directory
print(path.glob(here("../glob/third_party/c/README/*")))     # a file, not a directory
