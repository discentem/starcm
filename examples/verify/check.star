load("starcm", "here", "verify")

for f in ["agent.pkg", "unsigned.pkg"]:
    v = verify(here(f))
    print("%s: verified=%s key_id=%s" % (f, v.verified, v.key_id))
    if not v.verified:
        print("  error: %s" % v.error)
