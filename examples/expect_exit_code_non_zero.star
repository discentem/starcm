load("shellout", "exec")
a = exec(
    name               = "explicitly exit 2",
    cmd                = "sh", 
    args               = ["-c", "echo 'hello' > /dev/null; exit 2"],
    expected_exit_code = 2,
    live_output        = True
)
print(a)