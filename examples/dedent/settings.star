load("starcm", "dedent", "file", "vars")

def agent_settings(channel):
    # Indented with the code; dedent() takes the indentation off.
    return dedent("""
        server = https://config.example.com
        channel = %s
        [logging]
            level = info
        """ % channel)

r = file(label = "agent settings", path = vars["managed_dir"] + "/agent.conf", content = agent_settings("beta"))
print("changed=%s" % r.changed)
