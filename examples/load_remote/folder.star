load("starcm", "load_remote")

# folder = True loads a folder; its url ends in /. The signature covers the folder's
# manifest, which pins every file in it, so main.star, lib/settings.star and
# templates/motd.jinja need no .sig of their own. module is the .star file
# in the folder to run.
agent = load_remote(
    url       = "http://localhost:8000/examples/folders/agent/",
    folder    = True,
    module    = "main.star",
    signature = "http://localhost:8000/examples/folders/agent/starcm-manifest.json.sig",
    label     = "agent config",
).value
agent["configure"]()
