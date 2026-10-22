# date

[← All functions](../README.md)

Use `date` to read the current time, formatted and broken into fields. It
reads the clock in Go, so it works without a shell (for example in WinPE),
and it carries its own timezone database, so `timezone` works on Windows
too.

## Syntax

```python
load("starcm", "date")

date(
    label    = "now",       # required
    format   = "rfc3339",
    timezone = "Local",
)
```

## Arguments

| Argument | Type | Default | Description |
| --- | --- | --- | --- |
| `format` | string | `"rfc3339"` | One of the names below, or any [Go time layout](https://pkg.go.dev/time#pkg-constants) such as `"2006-01-02"`. |
| `timezone` | string | `"Local"` | An IANA name such as `"UTC"` or `"Europe/Paris"`, or `"Local"` for the machine's timezone. An unknown name fails the run. |

| `format` name | Example |
| --- | --- |
| `rfc3339` | `2026-10-17T14:05:09Z` |
| `rfc3339nano` | `2026-10-17T14:05:09.123456789Z` |
| `rfc1123` | `Sat, 17 Oct 2026 14:05:09 UTC` |
| `rfc1123z` | `Sat, 17 Oct 2026 14:05:09 +0000` |
| `kitchen` | `2:05PM` |
| `datetime` | `2026-10-17 14:05:09` |
| `date` | `2026-10-17` |
| `time` | `14:05:09` |
| `unixdate` | `Sat Oct 17 14:05:09 UTC 2026` |
| `stamp` | `Oct 17 14:05:09` |
| `iso8601basic` | `20261017T140509Z` |

`date` also takes the [common arguments](../common-functionality.md#common-arguments): `label`
(required), `only_if`, `not_if`, `timeout` and
`what_if`. Only `only_if` and `not_if` affect it.

## Returns

A [starcm_result](../common-functionality.md#starcm_result) whose `message` is
the formatted time and whose `value` is a `date` struct:

| Field | Type | Description |
| --- | --- | --- |
| `formatted` | string | The time in `format` |
| `rfc3339` | string | The time in RFC 3339, whatever `format` is |
| `unix`, `unix_nano` | int | Seconds / nanoseconds since 1970-01-01 UTC |
| `year`, `month`, `day` | int | Calendar date (`month` is 1-12) |
| `hour`, `minute`, `second` | int | Time of day |
| `weekday` | string | `"Monday"` ... `"Sunday"` |
| `yearday` | int | Day of the year, 1-366 |
| `timezone` | string | The timezone name, e.g. `"Asia/Tokyo"` or `"Local"` |
| `zone` | string | The zone abbreviation, e.g. `"JST"` |
| `offset_seconds` | int | Offset from UTC in seconds |

`success` is `True` and `changed` is `False`.

## Examples

The time changes on every run, so most expected output here is a regular
expression. Each example runs in a scratch copy of
[examples/date/](../../examples/date).

```scrut
$ cp -R "$TESTDIR/../../examples/date/." "$TMPDIR" && cd "$TMPDIR" && managed="$(pwd -W 2>/dev/null || pwd)"
```

### The current time

`TZ=UTC` pins the local timezone for this example.

```python
# examples/date/now.star
load("starcm", "date")

r = date(label = "now")
print("formatted: %s" % r.value.formatted)
print("message:   %s" % r.message)
```

```scrut
$ TZ=UTC starcm now.star
formatted: \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z (regex)
message:   \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z (regex)
```

### Named formats and Go layouts

```python
# examples/date/formats.star
load("starcm", "date")

for fmt in ["date", "datetime", "kitchen", "iso8601basic", "2006/01/02"]:
    print("%s: %s" % (fmt, date(label = fmt, format = fmt, timezone = "UTC").value.formatted))
```

```scrut
$ starcm formats.star
date: \d{4}-\d{2}-\d{2} (regex)
datetime: \d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} (regex)
kitchen: \d{1,2}:\d{2}[AP]M (regex)
iso8601basic: \d{8}T\d{6}Z (regex)
2006/01/02: \d{4}/\d{2}/\d{2} (regex)
```

### Fields in another timezone

```python
# examples/date/fields.star
load("starcm", "date")

d = date(label = "tokyo", timezone = "Asia/Tokyo").value
print("timezone=%s zone=%s offset_seconds=%d" % (d.timezone, d.zone, d.offset_seconds))
print("year=%d month=%d day=%d weekday=%s" % (d.year, d.month, d.day, d.weekday))
print("unix=%d" % d.unix)
```

```scrut
$ starcm fields.star
timezone=Asia/Tokyo zone=JST offset_seconds=32400
year=\d{4} month=\d{1,2} day=\d{1,2} weekday=\w+day (regex)
unix=\d+ (regex)
```

### Record when a run happened

```python
# examples/date/stamp_file.star
load("starcm", "date", "file", "vars")

today = date(label = "today", format = "date", timezone = "UTC").value.formatted
r = file(label = "last run", path = vars["managed_dir"] + "/last-run.txt", content = today + "\n")
print("changed=%s" % r.changed)
```

```scrut
$ starcm --var managed_dir="$managed" stamp_file.star && test "$(cat last-run.txt)" = "$(date -u +%Y-%m-%d)" && echo "matches the system clock"
changed=True
matches the system clock
```

### Several formats and timezones at once

`TZ=UTC` pins the local timezone for this example.

```python
# examples/date/date.star
load("starcm", "date")
load("starcm", "write")

now = date(label = "current time").value
write("local: " + now.formatted, label = "print local")

utc = date(label = "utc date", format = "date", timezone = "UTC").value
write("utc date: %s (%s, day %d of the year)" % (utc.formatted, utc.weekday, utc.yearday), label = "print utc")

la = date(label = "custom layout", format = "Mon Jan 2 15:04 MST", timezone = "America/Los_Angeles").value
write("los angeles: " + la.formatted, label = "print la")
```

```scrut
$ TZ=UTC starcm date.star
local: \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z (regex)
utc date: \d{4}-\d{2}-\d{2} \((Mon|Tues|Wednes|Thurs|Fri|Satur|Sun)day, day \d{1,3} of the year\) (regex)
los angeles: (Mon|Tue|Wed|Thu|Fri|Sat|Sun) (Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) \d{1,2} \d{2}:\d{2} P[DS]T (regex)
```

## Errors and edge cases

These are here mostly as tests: CI runs them to check error messages and
behaviour that the examples above don't show.

### An unknown timezone stops the run

```python
# examples/date/invalid_timezone.star
load("starcm", "date")

# An unknown IANA timezone stops the run with an error instead of
# returning a failed result.
date(label = "bad timezone", timezone = "Not/AZone")
```

```scrut
$ starcm invalid_timezone.star 2>&1
Traceback (most recent call last):
  invalid_timezone.star:5:5: in <toplevel>
Error in date: invalid timezone "Not/AZone": unknown time zone Not/AZone
[1]
```

## See also

