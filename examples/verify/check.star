load("starcm", "verify")

for f in ["agent.pkg", "unsigned.pkg"]:
    r = verify(f, label = "check " + f)
    print("%s: success=%s key=%s" % (f, r.success, r.value))
    if not r.success:
        print("  error: %s" % r.error)
