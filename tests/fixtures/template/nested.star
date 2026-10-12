load("starcm", "template", "write")

r = template(
    label    = "nested",
    template = "nested.tpl",
    what_if  = True,
    data     = {
        "user": {
            "name": "ada",
            "address": {"city": "london"},
            "pets": [{"kind": "cat", "name": "tom"}, {"kind": "dog", "name": "rex"}],
        },
        "env": {"ZED": "1", "ALPHA": "2"},
        "point": (3, 4),
        "nothing": None,
        "greeting": write("hello", label = "greeting", end = ""),
    },
)
print("")
write(r.message, label = "show render", end = "")
