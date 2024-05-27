'''
'''
load("shellout/shellout.star", "cmd")
load("shellout", "exec")
# # import a function defined in .star file
print(cmd())

a = exec(
    name               = "explicitly exit 2",
    cmd                = "sh", 
    args               = ["-c", "exit 2"],
    expected_exit_code = 2,
    # not_if = True
)
print(a)

exec(
    name               = "ping google 5 times",
    cmd                = "ping", 
    args               = ["-n", "google.com"],
    timeout            = "3s",
)
