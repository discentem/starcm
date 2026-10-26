load("starcm", "date")

for fmt in ["date", "datetime", "kitchen", "iso8601basic", "2006/01/02"]:
    print("%s: %s" % (fmt, date(format = fmt, timezone = "UTC").formatted))
