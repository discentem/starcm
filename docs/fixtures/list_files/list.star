load("starcm", "list_files")

print(list_files("config", label = "everything").value)
print(list_files("config", pattern = "*.star", label = "star files").value)
print(list_files(dir = "config/lib", pattern = "w*", label = "windows only").value)
