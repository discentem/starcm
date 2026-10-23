package base

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/discentem/starcm/libraries/paths"
	"go.starlark.net/starlark"
)

// Action is what a starcm function does. base checks a call's arguments
// once and passes them to Execute as a FunctionInstance, so the function
// never re-reads args and kwargs, and positional and keyword arguments
// behave the same for every function. See docs/design/paths-and-arguments.md.
type Action interface {
	Execute(inst *FunctionInstance) (*Result, error)
}

// FunctionInstance is one use of a starcm function: its arguments, already
// checked and parsed into the types the function declared with Arg,
// OptionalArg or OptionalArgAllowNone, plus the call's label and where it
// was written.
type FunctionInstance struct {
	Ctx    context.Context
	Label  string
	Thread *starlark.Thread
	// Dir is the directory of the .star file the call is written in, or ""
	// when it can't be told. Bundled resources resolve against it.
	Dir string
	// WhatIf is the what_if common argument: show what would change
	// without changing it.
	WhatIf bool

	// origin is where the .star file the call is written in runs from. A
	// single module loaded with load_remote runs from the store, where
	// nothing ships next to it, so relative resource paths can't resolve; in
	// a fetched folder they resolve, but only inside it.
	origin paths.Origin

	fn     string         // the function's name, for errors
	values map[string]any // name -> pointer to the parsed value (*T)
	given  map[string]bool
}

// Given reports whether the call passed argument name. For arguments declared
// with OptionalArgAllowNone, passing None counts as not passing it.
func (c *FunctionInstance) Given(name string) bool { return c.given[name] }

// Get returns argument name as the type T it was declared with, and whether
// the call passed it. For an omitted optional argument it returns T's zero
// value and false. Asking for a name that wasn't declared, or with a
// different type, is a programming error and panics.
func Get[T any](c *FunctionInstance, name string) (T, bool) {
	raw, ok := c.values[name]
	if !ok {
		panic(fmt.Sprintf("base.Get: %s has no argument %q", c.fn, name))
	}
	p, ok := raw.(*T)
	if !ok {
		var zero T
		panic(fmt.Sprintf("base.Get: %s argument %q is %T, not %T", c.fn, name, raw, &zero))
	}
	return *p, c.Given(name)
}

// AbsPath returns string argument name as an absolute path, with a leading ~
// expanded. A relative path is an error. The bool reports whether the call
// passed the argument; an omitted optional argument returns "", false, nil.
func (c *FunctionInstance) AbsPath(name string) (string, bool, error) {
	s, given := Get[string](c, name)
	if !given {
		return "", false, nil
	}
	p, err := paths.Abs(s)
	if err != nil {
		return "", true, c.argErr(name, err)
	}
	return p, true, nil
}

// ResourcePath returns string argument name as a path to a read-only file
// bundled with the code: a relative path resolves next to the .star file the
// call is written in.
func (c *FunctionInstance) ResourcePath(name string) (string, bool, error) {
	s, given := Get[string](c, name)
	if !given {
		return "", false, nil
	}
	if expanded, err := paths.Expand(s); err == nil && !filepath.IsAbs(expanded) {
		if !c.origin.Shipped() {
			return "", true, c.argErr(name, fmt.Errorf("%q is relative, but this .star file was loaded with load_remote and runs from starcm's store, where nothing ships next to it: fetch the file with download() or pass its absolute path", s))
		}
		if c.Dir != "" {
			p, err := c.origin.Resolve(c.Dir, expanded)
			if err != nil {
				return "", true, c.argErr(name, err)
			}
			return p, true, nil
		}
	}
	p, err := paths.Resource(c.Dir, s)
	if err != nil {
		return "", true, c.argErr(name, err)
	}
	return p, true, nil
}

func (c *FunctionInstance) argErr(name string, err error) error {
	return fmt.Errorf("%s(label=%q): %s: %w", c.fn, c.Label, name, err)
}

// checkRequired reports the first argument declared with Arg (no ? suffix)
// that the call didn't pass. UnpackArgs alone doesn't enforce this: once a
// parameter is optional, it treats every later one as optional too. name
// prefixes the error the way base names UnpackArgs errors.
func (c *FunctionInstance) checkRequired(name string, args []ArgPair) error {
	for _, a := range args {
		if !strings.HasSuffix(a.Key, "?") && !c.Given(a.Key) {
			return fmt.Errorf("%s: missing argument for %s", name, a.Key)
		}
	}
	return nil
}

// argName strips UnpackArgs' optional suffixes from an ArgPair key.
func argName(key string) string {
	return strings.TrimRight(key, "?")
}

// newInstance records the parsed targets and which arguments the call passed.
// names lists every key given to UnpackArgs, in order, so positional
// argument i fills names[i].
func newInstance(ctx context.Context, fn, label string, whatIf bool, thread *starlark.Thread, names []string, targets map[string]any, args starlark.Tuple, kwargs []starlark.Tuple) *FunctionInstance {
	noneIsUnset := map[string]bool{}
	for _, key := range names {
		if strings.HasSuffix(key, "??") {
			noneIsUnset[argName(key)] = true
		}
	}
	given := map[string]bool{}
	mark := func(name string, v starlark.Value) {
		if v == starlark.None && noneIsUnset[name] {
			return
		}
		given[name] = true
	}
	for i, v := range args {
		if i < len(names) {
			mark(argName(names[i]), v)
		}
	}
	for _, kv := range kwargs {
		if k, ok := starlark.AsString(kv[0]); ok {
			mark(k, kv[1])
		}
	}
	dir, _ := paths.CallerDir(thread)
	file, _ := paths.CallerFile(thread)
	var origin paths.Origin
	if file != "" {
		origin = paths.OriginOf(thread, file)
	}
	return &FunctionInstance{
		origin: origin,
		Ctx:    ctx,
		Label:  label,
		Thread: thread,
		Dir:    dir,
		WhatIf: whatIf,
		fn:     fn,
		values: targets,
		given:  given,
	}
}
