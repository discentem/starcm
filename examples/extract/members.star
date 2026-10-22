load("starcm", "extract", "here", "vars")

r = extract(
    label   = "unpack just the tool",
    archive = here("files/tool-1.0.zip"),
    dest    = vars["managed_dir"] + "/bin-only",
    members = ["tool-1.0/tool"],
)
print("changed=%s" % r.changed)
for path in r.value:
    print(path.replace("\\", "/").split("/")[-2:])  # last two elements, for this example
