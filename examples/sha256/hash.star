load("starcm", "sha256", "here")

r = sha256(here("payload.txt"), label = "hash payload")
print("success=%s changed=%s" % (r.success, r.changed))
print(r.value)
