load("starcm", "here", "path")

# * matches in any segment, so this looks one directory deep.
for m in path.glob(here("third_party/*/go.mod")):
    print(m)

# Directories match too.
print(len(path.glob(here("third_party", "*"))))
