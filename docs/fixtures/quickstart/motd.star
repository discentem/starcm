load("starcm", "exec", "file")

motd = file(
    label   = "message of the day",
    path    = "motd.txt",
    content = "Managed by starcm. Changes will be overwritten.\n",
)
print("motd: changed=%s" % motd.changed)

exec(
    label   = "show motd",
    cmd     = "cat",
    args    = ["motd.txt"],
    only_if = motd.changed,
    live_output = True,
)
