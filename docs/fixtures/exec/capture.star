load("starcm", "exec")

r = exec(label = "say hello", cmd = "echo", args = ["hello", "world"])
print(r)
