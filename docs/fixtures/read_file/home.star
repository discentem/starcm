load("starcm", "read_file")

print(read_file("~/notes.txt", label = "read from home").value.strip())
