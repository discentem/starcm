load("starcm", "date")
load("starcm", "write")

# "return" is a Starlark keyword, so read the payload with getattr().
now = getattr(date(label = "current time"), "return")
write("local: " + now.formatted, label = "print local")

utc = getattr(date(label = "utc date", format = "date", timezone = "UTC"), "return")
write("utc date: %s (%s, day %d of the year)" % (utc.formatted, utc.weekday, utc.yearday), label = "print utc")

la = getattr(date(label = "custom layout", format = "Mon Jan 2 15:04 MST", timezone = "America/Los_Angeles"), "return")
write("los angeles: " + la.formatted, label = "print la")
