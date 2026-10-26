load("starcm", "here", "path", "sign", "vars")

for f in path.list(here(), pattern = "*.star"):
    # Only this folder's configuration files, not this script or subfolders.
    if f == "sign_all.star" or "/" in f:
        continue
    r = sign(here(f), key = vars["key"], expires = "2099-01-01", label = "sign " + f)
    print("%s: changed=%s key=%s" % (f, r.changed, r.value))
