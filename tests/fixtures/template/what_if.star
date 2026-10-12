load("starcm", "template")

r = template(
    label    = "preview",
    template = "hello.tpl",
    data     = {"name": "world", "age": 42},
    what_if  = True,
)
print("success=%s changed=%s" % (r.success, r.changed))
print(r.message)
