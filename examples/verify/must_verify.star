load("starcm", "exec", "verify", "here")

check = verify(here("agent.pkg"), label = "check agent signature")
if not check.success:
    fail(check.error)
exec(label = "install agent", cmd = "cat", args = ["agent.pkg"], live_output = True)
