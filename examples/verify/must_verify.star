load("starcm", "exec", "verify", "here")

check = verify(here("agent.pkg"))
if not check.verified:
    fail(check.error)
exec(label = "install agent", cmd = "cat", args = ["agent.pkg"], live_output = True)
