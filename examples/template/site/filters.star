load("starcm", "template", "write")

r = template(
    label    = "filters",
    template = "upper.tpl",
    what_if  = True,
    data     = {"name": "fleet", "items": ["mac", "win"]},
)
write(r.message, label = "show render", end = "")
