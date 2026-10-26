load("starcm", "date", "file", "vars")

today = date(format = "date", timezone = "UTC").formatted
r = file(label = "last run", path = vars["managed_dir"] + "/last-run.txt", content = today + "\n")
print("changed=%s" % r.changed)
