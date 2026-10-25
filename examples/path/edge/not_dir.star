load("starcm", "here", "path")

# A file where a directory should be: the path doesn't exist.
print(path.exists(here("../files/agent.conf/x")))
