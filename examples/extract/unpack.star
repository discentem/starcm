load("starcm", "extract")

r = extract(
    label   = "unpack tool",
    archive = "files/tool-1.0.zip",
    dest    = "opt/tool",
)
print("changed=%s files=%d" % (r.changed, len(r.value)))
