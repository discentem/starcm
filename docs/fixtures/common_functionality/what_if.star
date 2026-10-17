load("starcm", "template")

r = template(
    label    = "dry run",
    template = "sub/hello.tpl",
    data     = {"where": "a dry run"},
    what_if  = True,
)
print("changed=%s message=%r" % (r.changed, r.message))
