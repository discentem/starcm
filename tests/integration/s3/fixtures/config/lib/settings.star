load("starcm", "dedent", "file")

def apply():
    """Writes the agent's settings file and reports whether it changed."""
    r = file(
        label       = "agent settings",
        path        = "managed/agent.conf",
        content     = dedent("""
            server = https://config.example.com
            channel = stable
            """),
        create_dirs = True,
    )
    print("agent settings: changed=%s" % r.changed)
