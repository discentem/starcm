load("starcm", "template")

template(label = "both", template = "hello.tpl", source = "Hello {{ name }}", data = {"name": "x"}, what_if = True)
