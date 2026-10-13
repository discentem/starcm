load("starcm", "list_files")

print(list_files("tree", label = "everything").value)
print(list_files(dir = "tree", pattern = "*.star", label = "star files").value)
print(list_files("tree/lib", label = "subdir").value)
