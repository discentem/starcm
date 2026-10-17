load("starcm", "sha256")

r = sha256("payload.txt", label = "hash payload")
print("success=%s changed=%s" % (r.success, r.changed))
print(r.value)
