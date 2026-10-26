load("starcm", "date")

d = date(timezone = "Asia/Tokyo")
print("timezone=%s zone=%s offset_seconds=%d" % (d.timezone, d.zone, d.offset_seconds))
print("year=%d month=%d day=%d weekday=%s" % (d.year, d.month, d.day, d.weekday))
print("unix=%d" % d.unix)
