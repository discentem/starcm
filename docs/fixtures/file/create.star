load("starcm", "file")

r = file(
    label   = "agent settings",
    path    = "agent.conf",
    content = "server = https://config.example.com\n",
)
print(r)
