load("shellout", "exec")
a = exec(
    name               = "ping google a few times",
    cmd                = "ping", 
    args               = ["-n", "google.com"],
    timeout            = "6s",
    live_output        = True
)
print(a)
