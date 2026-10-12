load("starcm", "date")
load("starcm", "write")

now = date(label = "current time").value
write("local: " + now.formatted, label = "print local")

utc = date(label = "utc date", format = "date", timezone = "UTC").value
write("utc date: %s (%s, day %d of the year)" % (utc.formatted, utc.weekday, utc.yearday), label = "print utc")

la = date(label = "custom layout", format = "Mon Jan 2 15:04 MST", timezone = "America/Los_Angeles").value
write("los angeles: " + la.formatted, label = "print la")
