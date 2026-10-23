load("starcm", "template", "vars")
load("lib/settings.star", "SETTINGS")

def configure():
    """Writes the message of the day from the template in this folder."""
    r = template(
        label       = "motd",
        template    = "templates/motd.jinja",  # ships in this folder
        data        = SETTINGS,
        destination = vars["managed_dir"] + "/motd",
    )
    print("motd: changed=%s" % r.changed)
