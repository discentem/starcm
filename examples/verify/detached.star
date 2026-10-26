load("starcm", "verify", "here")

v = verify(here("agent.pkg"), signature = here("sigs/agent.pkg.sig"))
print("verified=%s key_id=%s expires=%s" % (v.verified, v.key_id, v.expires))
