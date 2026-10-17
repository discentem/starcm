load("starcm", "vars", "write")

write("hello, %s" % vars.get("name", "world"), label = "greet")
