load("starcm", "load_dynamic", "vars")

platform = vars.get("platform", "macos")
r = load_dynamic("platforms/%s.star" % platform, label = "load %s settings" % platform)
print("success=%s changed=%s" % (r.success, r.changed))
print(sorted(r.value.keys()))
r.value["configure"]()
