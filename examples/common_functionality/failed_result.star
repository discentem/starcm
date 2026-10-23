load("starcm", "exec", "write")

r = exec(label = "check agent", cmd = "sh", args = ["-c", "echo not installed; exit 3"])

# A result that didn't succeed stops the run when it's passed on...
write(r, label = "skipped", only_if = False)  # ...unless the call is skipped anyway
write(r, label = "report")
