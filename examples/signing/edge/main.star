load("starcm", "load_dynamic", "vars", "write")
load("lib.star", "greet")

write(greet(vars.get("name", "nobody")), label = "greet")
load_dynamic("dyn.star", label = "load dyn.star")
