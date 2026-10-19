load("starcm", "file")

r = file(
    label   = "agent settings",
    path    = "agent.conf",
    content = "server = https://config.example.com\nchannel = beta\n",
)
print("changed=%s" % r.changed)
