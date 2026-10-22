load("starcm", "json", "read_file", "here")

settings = json.decode(read_file(path = here("settings.json"), label = "read settings").value)
print("channel=%s hosts=%s" % (settings["channel"], settings["hosts"]))
