load("starcm", "write", "vars")

env = vars.get("env", "dev")
write("deploying to production", label = "prod banner", only_if = env == "prod")
write("not production: %s" % env, label = "dev banner", not_if = env == "prod")
