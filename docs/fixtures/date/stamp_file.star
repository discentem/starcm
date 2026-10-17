load("starcm", "date", "file")

today = date(label = "today", format = "date", timezone = "UTC").value.formatted
r = file(label = "last run", path = "last-run.txt", content = today + "\n")
print("changed=%s" % r.changed)
