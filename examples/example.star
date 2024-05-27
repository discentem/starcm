'''
'''
load("shellout/shellout.star", "cmd")
load("shellout", "exec")
# # import a function defined in .star file
print(cmd())

exec(
    name               = "ping google 5 times",
    cmd                = "ping", 
    args               = ["-n", "google.com"],
    timeout            = "3s",
    live_output        = True
)

a = exec(
    name               = "explicitly exit 2",
    cmd                = "sh", 
    args               = ["-c", "exit 2"],
    expected_exit_code = 2,
    live_output        = True
    # not_if = True
)
# print(a)
