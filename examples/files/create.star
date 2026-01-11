load("starcm", "file")
load("starcm", "write")

file_result = file(
    label = "Create example.txt file",
    action = "create",
    path = "~/example.txt",
    content = "This is an example file created by starcm.\n",
)

write(
    file_result,
    label   = "Print file() result",
    only_if = file_result.success,
)