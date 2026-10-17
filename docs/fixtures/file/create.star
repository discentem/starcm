load("starcm", "file")

r = file(
    label   = "hosts allow list",
    path    = "hosts.allow",
    content = "sshd: 10.0.0.0/8\n",
)
print(r)
