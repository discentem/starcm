load("starcm", "template", "write")
load("stdlib", "struct")

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
        "greeting": struct(value = "hello", success = True),
    },
)
write(r.message, label = "show render", end = "")
