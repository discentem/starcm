load("starcm", "file")

print(
    file(
        label = "Create example.txt file",
        action = "create",
        path = "example.txt",
        content = "This is an example file created by starcm.\n",
    )
)