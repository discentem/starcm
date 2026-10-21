load("starcm", "extract")

# The format comes from the extension: .zip, .tar.gz or .tgz.
r = extract("files/tool-1.0.tar.gz", dest = "opt/tool-tar", label = "unpack tool")
print("changed=%s files=%d" % (r.changed, len(r.value)))
