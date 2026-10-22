load("starcm", "here")
load("../lib/render.star", "bundled_banner")

root = here("..")

def show(path):
    # Print paths relative to the example, with / even on Windows.
    return path[len(root) + 1:].replace("\\", "/")

print(show(here("motd.jinja")))  # next to this file
print(show(bundled_banner()))    # next to lib/render.star, where its here() call is
