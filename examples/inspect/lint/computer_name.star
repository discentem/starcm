load("starcm", "exec", "vars")

name = "%s-%s" % (vars["team"], vars["serial"])
exec(label = "set computer name", cmd = "scutil", args = ["--set", "ComputerName", name])
