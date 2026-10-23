load("starcm", "exec", "here", "template", "vars")
load("lib/settings.star", "SETTINGS")

def apply():
    """Renders the agent's settings from the template next to this file and
    runs the version script, both from the folder in the store."""
    r = template(
        label       = "agent settings",
        template    = "templates/agent.conf.jinja",
        data        = SETTINGS,
        destination = vars["managed_dir"] + "/agent-folder.conf",
    )
    print("agent settings: changed=%s" % r.changed)
    exec(label = "agent version", cmd = "sh", args = [here("scripts", "version.sh")], live_output = True)
