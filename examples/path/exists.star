load("starcm", "here", "path")

print(path.exists(here("files/agent.conf")))     # a file
print(path.exists(here("files")))                # a directory
print(path.exists(here("files/missing.conf")))   # nothing there
