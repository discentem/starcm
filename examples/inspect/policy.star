load("starcm", "here", "inspect", "path")

ALLOWED = ["download", "here", "load_dynamic"]

for f in path.list(here(), pattern = "*.star"):
    # Only this folder's configuration files, not this script or subfolders.
    if f == "policy.star" or "/" in f:
        continue
    report = inspect(here(f))
    bad = [c["function"] for c in report["calls"] if c["function"] not in ALLOWED]
    print("%s: %s" % (f, "ok" if not bad else "uses %s" % ", ".join(bad)))
