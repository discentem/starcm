# date()

Scrut tests for the `date` module. The current time changes on every run, so
most lines match with `(regex)`. `TZ=UTC` pins the local timezone.

## Formats and timezones

```scrut
$ TZ=UTC starcm examples/date/date.star
"local: \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z" (regex)
"utc date: \d{4}-\d{2}-\d{2} \((Mon|Tues|Wednes|Thurs|Fri|Satur|Sun)day, day \d{1,3} of the year\)" (regex)
"los angeles: (Mon|Tue|Wed|Thu|Fri|Sat|Sun) (Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) \d{1,2} \d{2}:\d{2} P[DS]T" (regex)
```

## Agrees with the system clock

```scrut
$ TZ=UTC starcm examples/date/date.star | grep -c "utc date: $(date -u +%Y-%m-%d) "
1
```

## Invalid timezone fails the run

```scrut
$ starcm examples/date/invalid_timezone.star 2>&1
Traceback (most recent call last):
  examples/date/invalid_timezone.star:5:5: in <toplevel>
Error in date: invalid timezone "Not/AZone": unknown time zone Not/AZone
[1]
```
