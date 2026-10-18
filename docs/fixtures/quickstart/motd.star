load("starcm", "exec", "file", "temp_dir")

# A starcm-quickstart folder in the system's temporary directory.
path = temp_dir("starcm-quickstart", "motd.txt")

motd = file(
    label       = "message of the day",
    path        = path,
    content     = "Managed by starcm. Changes will be overwritten.\n",
    create_dirs = True,
)
print("motd: changed=%s" % motd.changed)

exec(
    label   = "show motd",
    cmd     = "cat",
    args    = [path],
    only_if = motd.changed,
    live_output = True,
)
