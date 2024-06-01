load("shellout", "exec")

a = exec(
    name               = "explicitly exit 2",
    cmd                = "sh", 
    args               = ["-c", "echo 'we expect to exit 2'; exit 2"],
    expected_exit_code = 2,
)

def print_output_if_not_none(x):
    if x != None:
        print(x.output)

b = exec(
    name = "party??",
    cmd  = "sh",
    args = ["-c", "echo 'party!'"],
    only_if = a.success == True,
)

def print_results():
    for x in [a, b]:
        print_output_if_not_none(x)

print_results()

