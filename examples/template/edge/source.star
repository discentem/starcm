load("starcm", "template", "write")

r = template(label = "inline", source = "Hello {{ name | capitalize }}.\n", data = {"name": "world"}, what_if = True)
write(r.message, label = "show", end = "")
