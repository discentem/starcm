load("starcm", "template", "write")

render = template(
    name = "hello world template",
    template = "hello_world.tpl",
    data = {
        "name": "world",
        "age": 42,
    },
    what_if = True
)
write(name="render hello_world.tpl", str=render.output)
