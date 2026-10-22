load("starcm", "here", "verify")

for f in ["agent.pkg", "unsigned.pkg"]:
    r = verify(here(f), label = "check " + f)
    print("%s: success=%s key=%s" % (f, r.success, r.value))
    if not r.success:
        print("  error: %s" % r.error)
