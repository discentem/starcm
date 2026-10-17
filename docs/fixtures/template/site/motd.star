load("starcm", "template")

r = template(
    label       = "motd",
    template    = "motd.tpl",
    destination = "motd.txt",
    data        = {
        "hostname":    "build-01",
        "maintenance": "Sundays 02:00 UTC",
        "admins":      ["ada", "grace"],
    },
)
print("changed=%s" % r.changed)
