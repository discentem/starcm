load("starcm", "exec")
a = exec(
    name               = "ping apple.com",
    cmd                = "ping", 
    args               = ["-n", "apple.com"],
    live_output        = True
)
print(a)