load("starcm", "template", "write")

# what_if renders the template without writing a destination file; the
# rendered text comes back in the result's message, ending in the template's
# own trailing newline (hence end = "").
render = template(
    label = "hello world template",
    template = "hello_world.tpl",
    data = {
        "name": "world",
        "age": 42,
    },
    what_if = True
)
write(render.message, label = "render hello_world.tpl", end = "")
