load("starcm", "extract", "here", "vars")

# The format comes from the extension: .zip, .tar.gz or .tgz.
r = extract(here("files/tool-1.0.tar.gz"), dest = vars["managed_dir"] + "/opt/tool-tar", label = "unpack tool")
print("changed=%s files=%d" % (r.changed, len(r.value)))
