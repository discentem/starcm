load("starcm", "inspect", "list_files")

ALLOWED = ["download", "load_dynamic"]

for f in list_files(".", pattern = "*.star", label = "find config").value:
    # Only this folder's configuration files, not this script or subfolders.
    if f == "policy.star" or "/" in f:
        continue
    report = inspect(f, label = "inspect " + f).value
    bad = [c["function"] for c in report["calls"] if c["function"] not in ALLOWED]
    print("%s: %s" % (f, "ok" if not bad else "uses %s" % ", ".join(bad)))
