load("starcm", "extract", "here", "vars")

r = extract(
    label   = "unpack tool",
    archive = here("files/tool-1.0.zip"),
    dest    = vars["managed_dir"] + "/opt/tool",
)
print("changed=%s files=%d" % (r.changed, len(r.value)))
