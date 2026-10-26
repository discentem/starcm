load("starcm", "read_file", "here")

notes = read_file(here("notes.txt"))
print(repr(notes))
print(notes.splitlines())
