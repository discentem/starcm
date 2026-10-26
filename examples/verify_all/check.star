load("starcm", "here", "verify_all")

# Checks every file first, then stops the run once, listing each one that
# didn't verify.
for v in verify_all([here("agent.pkg"), here("unsigned.pkg"), here("other.pkg")]):
    print("%s: key_id=%s" % (v.path, v.key_id))
