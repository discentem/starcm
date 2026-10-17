load("starcm", "list_files", "sign", "vars")

for f in list_files(".", pattern = "*.star", label = "find config").value:
    if f == "sign_all.star":
        continue
    r = sign(f, key = vars["key"], expires = "2099-01-01", label = "sign " + f)
    print("%s: changed=%s key=%s" % (f, r.changed, r.value))
