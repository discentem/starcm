load("starcm", "file", "read_file", "temp_dir")

print(temp_dir())
print(temp_dir("starcm", "cache", "agent.json"))

note = temp_dir("starcm-docs", "note.txt")
r = file(label = "scratch note", path = note, content = "kept between runs\n", create_dirs = True)
print("changed=%s" % r.changed)
print(read_file(note, label = "read it back").value.strip())
