load("starcm", "exec")

exec(label = "show computer name", cmd = "scutil", args = ["--get", "ComputerName"])
