load("shellout", "exec")
a = exec(
    name               = "explicitly exit 2",
    cmd                = "sh", 
    args               = ["-c", "echo 'we expect to exit 2'; exit 2"],
    expected_exit_code = 2,
)


def party_if_success(fn):
    if fn.success == True:
        print("party!")
    else:
        print("no party :(")

print(a)
party_if_success(a)
