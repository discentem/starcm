package date

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"go.starlark.net/syntax"
)

var fixed = time.Date(2026, time.October, 11, 14, 30, 5, 0, time.UTC)

func run(t *testing.T, kwargs []starlark.Tuple) (*starlarkstruct.Struct, error) {
	t.Helper()
	a := &dateAction{now: func() time.Time { return fixed }}
	r, err := a.Run(context.Background(), "", "test", &starlark.Thread{}, nil, kwargs)
	if err != nil {
		return nil, err
	}
	require.True(t, r.Success)
	s, ok := r.Return.(*starlarkstruct.Struct)
	require.True(t, ok)
	return s, nil
}

func attr(t *testing.T, s *starlarkstruct.Struct, name string) starlark.Value {
	t.Helper()
	v, err := s.Attr(name)
	require.NoError(t, err)
	return v
}

func TestDateAction_Run(t *testing.T) {
	tests := []struct {
		name          string
		kwargs        []starlark.Tuple
		wantFormatted string
		wantHour      int
		wantErr       bool
	}{
		{
			name:          "default format in UTC",
			kwargs:        []starlark.Tuple{{starlark.String("timezone"), starlark.String("UTC")}},
			wantFormatted: "2026-10-11T14:30:05Z",
			wantHour:      14,
		},
		{
			name: "named format",
			kwargs: []starlark.Tuple{
				{starlark.String("format"), starlark.String("date")},
				{starlark.String("timezone"), starlark.String("UTC")},
			},
			wantFormatted: "2026-10-11",
			wantHour:      14,
		},
		{
			name: "custom go layout",
			kwargs: []starlark.Tuple{
				{starlark.String("format"), starlark.String("Mon Jan 2 15:04")},
				{starlark.String("timezone"), starlark.String("UTC")},
			},
			wantFormatted: "Sun Oct 11 14:30",
			wantHour:      14,
		},
		{
			name: "timezone conversion",
			kwargs: []starlark.Tuple{
				{starlark.String("format"), starlark.String("datetime")},
				{starlark.String("timezone"), starlark.String("America/Los_Angeles")},
			},
			wantFormatted: "2026-10-11 07:30:05",
			wantHour:      7,
		},
		{
			name:    "invalid timezone",
			kwargs:  []starlark.Tuple{{starlark.String("timezone"), starlark.String("Not/AZone")}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := run(t, tt.kwargs)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, starlark.String(tt.wantFormatted), attr(t, s, "formatted"))
			require.Equal(t, starlark.MakeInt(tt.wantHour), attr(t, s, "hour"))
			require.Equal(t, starlark.MakeInt64(fixed.Unix()), attr(t, s, "unix"))
			require.Equal(t, starlark.MakeInt(2026), attr(t, s, "year"))
		})
	}
}

// Exercises the builtin end to end through starlark, including base.Module's
// label handling and the starcm_result wrapper.
func TestDate_Starlark(t *testing.T) {
	ctx := context.Background()
	predeclared := starlark.StringDict{
		"date": starlark.NewBuiltin("date", newWithClock(ctx, func() time.Time { return fixed }).Function()),
	}
	src := `
d = date(label = "now", format = "date", timezone = "UTC")
formatted = d.value.formatted
success = d.success
weekday = d.value.weekday
`
	globals, err := starlark.ExecFileOptions(&syntax.FileOptions{}, &starlark.Thread{}, "test.star", src, predeclared)
	require.NoError(t, err)
	require.Equal(t, starlark.String("2026-10-11"), globals["formatted"])
	require.Equal(t, starlark.True, globals["success"])
	require.Equal(t, starlark.String("Sunday"), globals["weekday"])
}
