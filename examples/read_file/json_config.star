load("starcm", "json", "read_file")

settings = json.decode(read_file(path = "settings.json", label = "read settings").value)
print("channel=%s hosts=%s" % (settings["channel"], settings["hosts"]))
