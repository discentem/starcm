load("starcm", "dedent", "file", "vars")

def apply():
    """Writes the agent's settings file and reports whether it changed."""
    # A module loaded with load_remote runs from the store, where here()
    # isn't available, so the run says where managed files go.
    r = file(
        label       = "agent settings",
        path        = vars["managed_dir"] + "/agent.conf",
        content     = dedent("""
            server = https://config.example.com
            channel = stable
            """),
        create_dirs = True,
    )
    print("agent settings: changed=%s" % r.changed)
