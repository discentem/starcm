load("starcm", "exec")

exec(label = "live", cmd = "sh", args = ["-c", "echo streamed"], live_output = True)
