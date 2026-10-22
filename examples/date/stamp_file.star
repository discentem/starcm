load("starcm", "date", "file", "vars")

today = date(label = "today", format = "date", timezone = "UTC").value.formatted
r = file(label = "last run", path = vars["managed_dir"] + "/last-run.txt", content = today + "\n")
print("changed=%s" % r.changed)
