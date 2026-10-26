load("starcm", "here", "inspect", "path")

for f in path.list(here(), pattern = "*.star"):
    for c in inspect(here(f), label = "inspect " + f).value["calls"]:
        if c["function"] != "exec":
            continue
        for i in c.get("inputs", []):
            if "vars" in [u["function"] for u in i["uses"]]:
                print("%s:%d: exec's %s argument comes from vars" % (f, c["line"], i["arg"]))
