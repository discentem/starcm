load("starcm", "exec")

a = exec(
    name               = "ping apple a few times",
    cmd                = "ping", 
    args               = ["-n", "apple.com"],
    timeout            = "6s",
    live_output        = True
)
print(a)
