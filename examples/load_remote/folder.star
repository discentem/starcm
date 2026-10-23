load("starcm", "load_remote")

# A URL ending in / loads a folder. The signature covers the folder's
# manifest, which pins every file in it, so main.star, lib/settings.star and
# templates/motd.jinja need no .sig of their own. entry is the module to run.
agent = load_remote(
    url       = "http://localhost:8000/examples/folders/agent/",
    entry     = "main.star",
    signature = "http://localhost:8000/examples/folders/agent/starcm-manifest.json.sig",
    label     = "agent config",
).value
agent["configure"]()
