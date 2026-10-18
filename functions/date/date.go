package date

import (
	"context"
	"fmt"
	"time"
	// Windows hosts have no zoneinfo database, so embed Go's copy for timezone=.
	_ "time/tzdata"

	"github.com/discentem/starcm/functions/base"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
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

type dateAction struct {
	now func() time.Time
}

var _ base.Runnable = (*dateAction)(nil)

type parsedArgs struct {
	layout   string
	location *time.Location
}

func (a *dateAction) parseArgs(kwargs []starlark.Tuple) (*parsedArgs, error) {
	format, err := starlarkhelpers.FindValueInKwargsWithDefault(kwargs, "format", "rfc3339")
	if err != nil {
		return nil, err
	}
	layout, ok := namedLayouts[*format]
	if !ok {
		layout = *format
	}

	tz, err := starlarkhelpers.FindValueInKwargsWithDefault(kwargs, "timezone", "Local")
	if err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(*tz)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q: %w", *tz, err)
	}

	return &parsedArgs{layout: layout, location: loc}, nil
}

func (a *dateAction) Run(ctx context.Context, workingDirectory string, moduleName string, thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	parsed, err := a.parseArgs(kwargs)
	if err != nil {
		return nil, err
	}

	now := a.now().In(parsed.location)
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

	return &base.Result{
		Label:   moduleName,
		Message: &formatted,
		Success: true,
		Changed: false,
		Return:  ret,
	}, nil
}

// New returns the date() builtin. It reads the clock natively in Go, so it
// works anywhere starcm.exe does (including WinPE) without a shell.
//
//	d = date(label = "now", format = "rfc3339", timezone = "UTC")
//	d.value.formatted, d.value.year, d.value.unix, ...
func New(ctx context.Context) *base.Module {
	return newWithClock(ctx, time.Now)
}

func newWithClock(ctx context.Context, now func() time.Time) *base.Module {
	return base.NewModule(
		ctx,
		"date",
		[]base.ArgPair{
			base.OptionalArgAllowNone[string]("format"),
			base.OptionalArgAllowNone[string]("timezone"),
		},
		&dateAction{now: now},
	)
}
