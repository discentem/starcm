load("starcm", "date")

r = date(label = "now")
print("formatted: %s" % r.value.formatted)
print("message:   %s" % r.message)
