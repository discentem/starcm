load("starcm", "file", "vars")

r = file(
    label       = "agent config",
    path        = vars["managed_dir"] + "/etc/agent/agent.conf",
    content     = "server = https://config.example.com\n",
    mode        = 0o600,
    create_dirs = True,
)
print("success=%s changed=%s" % (r.success, r.changed))
