load("starcm", "file", "vars")

r = file(
    label   = "agent settings",
    path    = vars["managed_dir"] + "/agent.conf",
    content = """\
server = https://config.example.com
channel = beta
""",
)
print("changed=%s" % r.changed)
