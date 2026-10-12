load("starcm", "load_dynamic")

# Unlike load(), load_dynamic() can take a computed path.
for name in ["lib"]:
    print(load_dynamic("%s.star" % name, label = "load " + name).value["GREETING"])
