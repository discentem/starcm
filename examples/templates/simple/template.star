load("starcm", "template")
load("starcm", "write")

render = template(
    name = "example template",
    template = "example.tpl",
    key_vals = {
        "name": "world",
        "age": 42,
    }
)
write(render.output)
