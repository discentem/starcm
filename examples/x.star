load("shellout", "exec")
load("write", "write")

a = exec(
    name               = "explicitly exit 2",
    cmd                = "sh", 
    args               = ["-c", "echo 'we expect to exit 2'; exit 2"],
    expected_exit_code = 2,
)

write(
    name = "print_success",
    str = "a.success: %s" % (a.success),
    only_if = a.success == False
)

if a.success:
    write(
        name = "print_output",
        str = "a.output: %s #2" % (a.success),
    )

