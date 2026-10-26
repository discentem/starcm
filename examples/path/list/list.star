load("starcm", "here", "path")

print(path.list(here("config")))
print(path.list(here("config"), pattern = "*.star"))
print(path.list(dir = here("config/lib"), pattern = "w*"))
