load("starcm", "template", "write")

r = template(
    label    = "preview",
    template = "hello.tpl",
    data     = {"name": "world", "age": 42},
    what_if  = True,
)
print("success=%s changed=%s" % (r.success, r.changed))

# The rendered text keeps the template's trailing newline, so don't add one.
write(r.message, label = "show render", end = "")
