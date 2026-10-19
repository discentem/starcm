load("starcm", "load_dynamic", "vars", run = "exec")
load("lib.star", "greet")

run(label = "list", cmd = "ls", args = [])
load_dynamic("lib.star", label = "literal path")
load_dynamic(vars["next"], label = "computed path")
greet("ada")
