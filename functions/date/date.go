package date

import (
	"context"
	"time"

	// Windows hosts have no zoneinfo database, so embed Go's copy for timezone=.
	_ "time/tzdata"

	"github.com/discentem/starcm/functions/base"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// namedLayouts maps friendly format names to Go time layouts. Any other value
// passed as format is used directly as a Go layout (e.g. "2006-01-02").
var namedLayouts = map[string]string{
	"rfc3339":      time.RFC3339,
	"rfc3339nano":  time.RFC3339Nano,
	"rfc1123":      time.RFC1123,
	"rfc1123z":     time.RFC1123Z,
	"kitchen":      time.Kitchen,
	"datetime":     time.DateTime,
	"date":         time.DateOnly,
	"time":         time.TimeOnly,
	"unixdate":     time.UnixDate,
	"stamp":        time.Stamp,
	"iso8601basic": "20060102T150405Z0700",
}

type parsedArgs struct {
	layout   string
	location *time.Location
}

func parseArgs(inst *base.FunctionInstance) (*parsedArgs, error) {
	format, given := base.Get[string](inst, "format")
	if !given {
		format = "rfc3339"
	}
	layout, ok := namedLayouts[format]
	if !ok {
		layout = format
	}

	tz, given := base.Get[string](inst, "timezone")
	if !given {
		tz = "Local"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, inst.Errorf("invalid timezone %q: %w", tz, err)
	}

	return &parsedArgs{layout: layout, location: loc}, nil
}

// run returns the date struct for now, read from the clock.
func run(inst *base.FunctionInstance, clock func() time.Time) (starlark.Value, error) {
	parsed, err := parseArgs(inst)
	if err != nil {
		return nil, err
	}

	now := clock().In(parsed.location)
	formatted := now.Format(parsed.layout)
	zone, offset := now.Zone()

	ret := starlarkstruct.FromStringDict(starlark.String("date"), starlark.StringDict{
		"formatted":      starlark.String(formatted),
		"rfc3339":        starlark.String(now.Format(time.RFC3339)),
		"unix":           starlark.MakeInt64(now.Unix()),
		"unix_nano":      starlark.MakeInt64(now.UnixNano()),
		"year":           starlark.MakeInt(now.Year()),
		"month":          starlark.MakeInt(int(now.Month())),
		"day":            starlark.MakeInt(now.Day()),
		"hour":           starlark.MakeInt(now.Hour()),
		"minute":         starlark.MakeInt(now.Minute()),
		"second":         starlark.MakeInt(now.Second()),
		"weekday":        starlark.String(now.Weekday().String()),
		"yearday":        starlark.MakeInt(now.YearDay()),
		"timezone":       starlark.String(parsed.location.String()),
		"zone":           starlark.String(zone),
		"offset_seconds": starlark.MakeInt(offset),
	})

	return ret, nil
}

// New returns date(), a query for the current time. It reads the clock
// natively in Go, so it works anywhere starcm.exe does (including WinPE)
// without a shell.
//
//	d = date(format = "rfc3339", timezone = "UTC")
//	d.formatted, d.year, d.unix, ...
func New(ctx context.Context) *base.Query {
	return newWithClock(ctx, time.Now)
}

func newWithClock(ctx context.Context, clock func() time.Time) *base.Query {
	return base.NewQuery(
		ctx,
		"date",
		[]base.ArgPair{
			base.OptionalArgAllowNone[string]("format"),
			base.OptionalArgAllowNone[string]("timezone"),
		},
		func(inst *base.FunctionInstance) (starlark.Value, error) {
			return run(inst, clock)
		},
	)
}
