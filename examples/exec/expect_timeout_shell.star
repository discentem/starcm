load("starcm", "exec")
a = exec(
    name               = "sleep for 5s, timeout after 3",
    cmd                = "sleep", 
    args               = ["5"],
    timeout            = "3s",
    live_output        = True
)
print(a)