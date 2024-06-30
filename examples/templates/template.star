load("template", "template")
load("write", "write")

render = template(
    name = "example template",
    template = "example.tpl",
    key_vals = {
        "name": "world",
        "age": 42
    }
)
write(str=render.output)
