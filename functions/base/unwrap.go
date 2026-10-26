package base

import (
	"fmt"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// resultConstructor is the constructor name every starcm_result carries; see
// Result.ToStarlark.
var resultConstructor = starlark.String("starcm_result")

// project returns what a starcm_result stands for in one argument, and
// whether it counts as a failure that stops the run.
type project func(s *starlarkstruct.Struct) (v starlark.Value, failed bool)

// byValue is how most arguments read a result: as its value. One that didn't
// succeed is a failure.
func byValue(s *starlarkstruct.Struct) (starlark.Value, bool) {
	success, err := s.Attr("success")
	failed := err != nil || success != starlark.True
	value, err := s.Attr("value")
	if err != nil {
		return starlark.None, failed
	}
	return value, failed
}

// bySuccess is how conditions read a result: only_if = r means
// only_if = r.success. A result that didn't succeed is just False, never a
// failure, so not_if = check runs the step when check failed.
func bySuccess(s *starlarkstruct.Struct) (starlark.Value, bool) {
	if success, err := s.Attr("success"); err == nil && success == starlark.True {
		return starlark.True, false
	}
	return starlark.False, false
}

// projectionFor returns how argument name reads a result.
func projectionFor(name string) project {
	if name == "only_if" || name == "not_if" {
		return bySuccess
	}
	return byValue
}

// failedResult records the first result passed as an argument that didn't
// succeed. Its value still stands in for it, so the call can be checked and
// skipped by only_if or not_if before the failure stops the run.
type failedResult struct {
	arg    string
	label  string
	reason string
}

func (f *failedResult) err(fn string) error {
	return fmt.Errorf("%s: %s is the result of %q, which didn't succeed (%s); pass r.value to use it anyway", fn, f.arg, f.label, f.reason)
}

// unwrapArgs replaces every starcm_result in args and kwargs, including
// inside lists, tuples and dicts, with what it stands for in that argument
// (see projectionFor): its value, so extract(archive = zip) works without
// .value, or its success in only_if and not_if. names gives positional arguments their names for
// errors. Values that hold no result are returned as they are.
func unwrapArgs(names []string, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Tuple, []starlark.Tuple, *failedResult) {
	var failed *failedResult
	note := func(arg string, s *starlarkstruct.Struct) {
		if failed != nil {
			return
		}
		f := &failedResult{arg: arg}
		if v, err := s.Attr("label"); err == nil {
			f.label, _ = starlark.AsString(v)
		}
		if v, err := s.Attr("error"); err == nil {
			f.reason, _ = starlark.AsString(v)
		}
		if f.reason == "" || f.reason == "<nil>" {
			f.reason = "it was skipped or failed"
		}
		failed = f
	}

	var newArgs starlark.Tuple
	for i, v := range args {
		name := fmt.Sprintf("argument %d", i+1)
		if i < len(names) {
			name = argName(names[i])
		}
		if u, ok := unwrap(v, projectionFor(name), func(s *starlarkstruct.Struct) { note(name, s) }); ok {
			if newArgs == nil {
				newArgs = append(starlark.Tuple(nil), args...)
			}
			newArgs[i] = u
		}
	}
	if newArgs == nil {
		newArgs = args
	}

	var newKwargs []starlark.Tuple
	for i, kv := range kwargs {
		name, _ := starlark.AsString(kv[0])
		if u, ok := unwrap(kv[1], projectionFor(name), func(s *starlarkstruct.Struct) { note(name, s) }); ok {
			if newKwargs == nil {
				newKwargs = append([]starlark.Tuple(nil), kwargs...)
			}
			newKwargs[i] = starlark.Tuple{kv[0], u}
		}
	}
	if newKwargs == nil {
		newKwargs = kwargs
	}
	return newArgs, newKwargs, failed
}

// isResult reports whether v is a starcm_result.
func isResult(v starlark.Value) (*starlarkstruct.Struct, bool) {
	s, ok := v.(*starlarkstruct.Struct)
	return s, ok && s.Constructor() == resultConstructor
}

// unwrap returns v with every starcm_result in it replaced by what p says it
// stands for, and whether anything was replaced. Containers are copied only
// when something in them changed. onFailed is called for each result p
// counts as a failure.
func unwrap(v starlark.Value, p project, onFailed func(*starlarkstruct.Struct)) (starlark.Value, bool) {
	if s, ok := isResult(v); ok {
		value, failed := p(s)
		if failed {
			onFailed(s)
		}
		u, _ := unwrap(value, p, onFailed)
		return u, true
	}
	switch x := v.(type) {
	case *starlark.List:
		var out []starlark.Value
		for i := 0; i < x.Len(); i++ {
			if u, ok := unwrap(x.Index(i), p, onFailed); ok {
				if out == nil {
					out = make([]starlark.Value, x.Len())
					for j := 0; j < x.Len(); j++ {
						out[j] = x.Index(j)
					}
				}
				out[i] = u
			}
		}
		if out != nil {
			return starlark.NewList(out), true
		}
	case starlark.Tuple:
		var out starlark.Tuple
		for i, e := range x {
			if u, ok := unwrap(e, p, onFailed); ok {
				if out == nil {
					out = append(starlark.Tuple(nil), x...)
				}
				out[i] = u
			}
		}
		if out != nil {
			return out, true
		}
	case *starlark.Dict:
		changed := false
		items := x.Items()
		for i, kv := range items {
			if u, ok := unwrap(kv[1], p, onFailed); ok {
				items[i] = starlark.Tuple{kv[0], u}
				changed = true
			}
		}
		if changed {
			d := starlark.NewDict(len(items))
			for _, kv := range items {
				_ = d.SetKey(kv[0], kv[1])
			}
			return d, true
		}
	}
	return v, false
}

// skippedByKwargs reports whether kwargs skip the call with only_if = False
// or not_if = True, for when the arguments can't be unpacked to tell.
func skippedByKwargs(kwargs []starlark.Tuple) bool {
	for _, kv := range kwargs {
		name, _ := starlark.AsString(kv[0])
		b, ok := kv[1].(starlark.Bool)
		if !ok {
			continue
		}
		if (name == "only_if" && !bool(b)) || (name == "not_if" && bool(b)) {
			return true
		}
	}
	return false
}
