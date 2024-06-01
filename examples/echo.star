load("shellout", "exec")
exec(
    name               = "hello_from_starcm",
    cmd                = "echo", 
    args               = ["hello from starcm!"],
    timeout            = "3s",
    live_output        = True
)