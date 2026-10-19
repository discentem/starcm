load("starcm", "write")
load("stdlib", "struct")

app = struct(name = "agent", version = "1.2.0", ports = [443, 8443])
print(app.name, app.version, app.ports)
print(app)
write("%s %s" % (app.name, app.version), label = "use fields")
