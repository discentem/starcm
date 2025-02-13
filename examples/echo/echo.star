load("starcm", "exec")
exec(
    name               = "hello_from_echo_dot_star",
    cmd                = "echo", 
    args               = ["hello from echo.star!"],
    live_output        = True
)