load("starcm", "template")

def render(label, age):
    r = template(
        label       = label,
        template    = "hello.tpl",
        data        = {"name": "world", "age": age},
        destination = "hello.txt",
    )
    print("%s: success=%s changed=%s" % (label, r.success, r.changed))

render("first render", 42)
render("same data", 42)
render("new data", 43)
