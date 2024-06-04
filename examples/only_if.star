load("shellout", "exec")
load("write", "write")

a = exec(
    name               = "explicitly exit 2",
    cmd                = "sh", 
    args               = ["-c", "echo 'we expect to exit 2'; exit 2"],
    expected_exit_code = 2,
)

b = exec(
    name = "party??",
    cmd  = "sh",
    args = ["-c", "echo 'party!'"],
    only_if = a.success == False,
)

write(
    name = "print_success",
    str = "a.success: %s, b.success: %s" % (a.success, b.success),
    only_if = ((a.success == True) and (b.success == True)),
)

