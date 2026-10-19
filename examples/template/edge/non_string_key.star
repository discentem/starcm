load("starcm", "template")

template(label = "bad keys", template = "hello.tpl", what_if = True, data = {"name": "x", "age": {1: "one"}})
