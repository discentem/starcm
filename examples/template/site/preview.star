load("starcm", "template", "write")

r = template(
    label    = "preview agent.ini",
    template = "ini.tpl",
    what_if  = True,
    data     = {
        "sections": [
            {"name": "server", "settings": {"host": "config.example.com", "port": 443}},
            {"name": "logging", "settings": {"level": "info"}},
        ],
    },
)
print("changed=%s" % r.changed)
write(r.message, label = "show render", end = "")
