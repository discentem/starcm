load("starcm", "glob", "here")

print(glob(here("../third_party/*/Cargo.toml"), label = "no matches").value)
print(glob(here("../missing/*/go.mod"), label = "missing dir").value)
print(glob(here("../third_party/c/README/*"), label = "file, not dir").value)
