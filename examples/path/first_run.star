load("starcm", "file", "home_dir", "path")

marker = home_dir(".agent", "installed")
if path.exists(marker):
    print("already installed")
else:
    file(label = "mark installed", path = marker, content = "", create_dirs = True)
    print("first run: installed")
