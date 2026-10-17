load("starcm", "verify")

r = verify("agent.pkg", signature = "sigs/agent.pkg.sig", label = "check with a detached signature")
print("success=%s key=%s" % (r.success, r.value))
