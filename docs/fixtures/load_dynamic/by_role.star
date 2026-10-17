load("starcm", "load_dynamic", "vars")

role = vars.get("role", "web")
r = load_dynamic("roles/%s.star" % role, label = "load role %s" % role)
print("success=%s changed=%s" % (r.success, r.changed))
print(sorted(r.value.keys()))
r.value["configure"]()
