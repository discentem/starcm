load("starcm", "load_dynamic", "vars")

r = load_dynamic(
    "roles/db.star",
    label   = "database role",
    only_if = vars.get("with_db") == "yes",
)
if r.value:
    r.value["configure"]()
else:
    print("database role skipped")
