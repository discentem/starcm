load("starcm", "exec", "write")

today = exec(label = "print the date", cmd = "echo", args = ["2026-10-17"])

# Passing the result itself passes its value, so these two calls write the
# same thing. It works inside lists and dicts too.
write(today, label = "the result", end = "")
write(today.value, label = "its value", end = "")
