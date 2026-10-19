load("starcm", "file", "json")

settings = {"channel": "beta", "auto_update": True, "hosts": ["a", "b"]}
r = file(
    label   = "settings.json",
    path    = "settings.json",
    content = json.indent(json.encode(settings), indent = "  ") + "\n",
)
print("changed=%s" % r.changed)
