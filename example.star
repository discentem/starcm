'''
'''
load("shellout.star", "cmd")
load("shellout", "exec")
# # import a function defined in .star file
# print(cmd())

a = exec(
    name               = "explicitly exit 1",
    cmd                = "sh", 
    args               = ["-c", "exit 2"],
    expected_exit_code = 2,
)
print(a)