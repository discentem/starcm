load("starcm", "load_dynamic", "vars")

r = load_dynamic(
    "extras/developer_tools.star",
    label   = "developer tools",
    only_if = vars.get("developer") == "yes",
)
if r.value:
    r.value["configure"]()
else:
    print("developer tools skipped")
