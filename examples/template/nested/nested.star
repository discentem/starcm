load("starcm", "date", "template", "write")

# data can hold nested dicts, lists, tuples, None and structs (a result
# from another starcm function stands for its value). Templates reach into them with
# dots or brackets: server.host, users[0]["name"], now.formatted.
render = template(
    label = "render config.tpl",
    template = "config.tpl",
    data = {
        "server": {"host": "localhost", "port": 8080},
        "users": [
            {"name": "ada", "groups": ("admin", "dev"), "shell": "/bin/zsh"},
            {"name": "bob", "groups": ["dev"], "shell": None},
        ],
        "now": date(format = "date", timezone = "UTC"),
    },
    what_if = True,
)
write(render.message, label = "print config", end = "")
