load("starcm", "exec")
exec(
    name               = "hello_from_starcm",
    cmd                = "echo", 
    args               = ["hello from echo.star!"],
    live_output        = True
)