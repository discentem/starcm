load("starcm", "file")

r = file(
    label   = "hosts allow list",
    path    = "hosts.allow",
    content = "sshd: 10.0.0.0/8\nsshd: 192.168.0.0/16\n",
)
print("changed=%s" % r.changed)
