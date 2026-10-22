load("starcm", "list_files", "here")

print(list_files(here("config"), label = "everything").value)
print(list_files(here("config"), pattern = "*.star", label = "star files").value)
print(list_files(dir = here("config/lib"), pattern = "w*", label = "windows only").value)
