load("starcm", "load_dynamic")

lib = load_dynamic(module = "lib.star", label = "load lib by kwarg").value
print(lib["GREETING"])
