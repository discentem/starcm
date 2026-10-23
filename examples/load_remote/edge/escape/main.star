load("starcm", "here")

# Inside a fetched folder, here() and relative paths can't leave it.
print(here("..", "..", "secrets.txt"))
