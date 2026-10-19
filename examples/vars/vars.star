load("starcm", "vars")

print(vars)
print("channel=%s" % vars.get("channel", "stable"))
