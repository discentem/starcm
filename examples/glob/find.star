load("starcm", "glob", "here")

# * matches in any segment, so this looks one directory deep.
for m in glob(here("third_party/*/go.mod"), label = "find modules").value:
    print(m)

# Directories match too.
print(len(glob(here("third_party", "*"), label = "vendored dirs").value))
