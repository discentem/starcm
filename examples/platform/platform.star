load("starcm", "platform")

p = platform()
print("os=%s arch=%s" % (p.os, p.arch))
print("terraform_%s_%s.zip" % (p.os, p.arch))
