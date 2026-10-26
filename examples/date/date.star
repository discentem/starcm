load("starcm", "date")
load("starcm", "write")

now = date()
write("local: " + now.formatted, label = "print local")

utc = date(format = "date", timezone = "UTC")
write("utc date: %s (%s, day %d of the year)" % (utc.formatted, utc.weekday, utc.yearday), label = "print utc")

la = date(format = "Mon Jan 2 15:04 MST", timezone = "America/Los_Angeles")
write("los angeles: " + la.formatted, label = "print la")
