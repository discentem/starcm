load("starcm", "load_dynamic")

r = load_dynamic("lib.star", label = "load lib")
print("success=%s changed=%s" % (r.success, r.changed))
print(sorted(r.value.keys()))
print(r.value["greet"]("starcm"))
