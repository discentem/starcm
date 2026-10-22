load("starcm", "template")

def render(dest):
    # motd.jinja would have to ship next to this file, but in the store
    # nothing does.
    template(label = "motd", template = "motd.jinja", data = {}, destination = dest)
