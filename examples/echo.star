load("shellout", "exec")

a = exec(
    name = "echo_hello",
    cmd  = "echo", 
    args = ["hello world!"],
)
print(a)