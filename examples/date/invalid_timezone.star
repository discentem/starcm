load("starcm", "date")

# An unknown IANA timezone stops the run with an error instead of
# returning a failed result.
date(label = "bad timezone", timezone = "Not/AZone")
