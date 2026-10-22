load("starcm", "verify", "here")

r = verify(here("agent.pkg"), signature = here("sigs/agent.pkg.sig"), label = "check with a detached signature")
print("success=%s key=%s" % (r.success, r.value))
