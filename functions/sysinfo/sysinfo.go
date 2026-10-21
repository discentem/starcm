// Package sysinfo implements the plain helpers platform(), env() and
// home_dir(), which describe the machine starcm runs on.
package sysinfo

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mitchellh/go-homedir"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// Platform returns the platform() builtin. It returns a struct with string
// fields os and arch, named as Go's runtime.GOOS and runtime.GOARCH are:
//
//	platform()   # os = "darwin", arch = "arm64" on Apple silicon
func Platform() *starlark.Builtin {
	return starlark.NewBuiltin("platform", func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if len(args) > 0 || len(kwargs) > 0 {
			return nil, fmt.Errorf("%s: takes no arguments", b.Name())
		}
		return starlarkstruct.FromStringDict(starlark.String("platform"), starlark.StringDict{
			"os":   starlark.String(runtime.GOOS),
			"arch": starlark.String(runtime.GOARCH),
		}), nil
	})
}

// Env returns the env() builtin. It returns the value of the environment
// variable name, or default when the variable isn't set. A variable that is
// set to "" returns "", not default. default is None when omitted, so an
// unset variable returns None.
//
//	env("REGION", default = "us-east-1")
func Env() *starlark.Builtin {
	return starlark.NewBuiltin("env", func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var name string
		var fallback starlark.Value = starlark.None
		if err := starlark.UnpackArgs(b.Name(), args, kwargs, "name", &name, "default?", &fallback); err != nil {
			return nil, err
		}
		if _, ok := fallback.(starlark.String); !ok && fallback != starlark.None {
			return nil, fmt.Errorf("%s: default: got %s, want string or None", b.Name(), fallback.Type())
		}
		if value, ok := os.LookupEnv(name); ok {
			return starlark.String(value), nil
		}
		return fallback, nil
	})
}

// HomeDir returns the home_dir() builtin. It returns the user's home
// directory joined with the string parts. A leading ~ in a path expands to
// the same directory, so home_dir() matches what file() does with "~".
//
//	home_dir(".config", "agent.toml")   # "/Users/me/.config/agent.toml"
func HomeDir() *starlark.Builtin {
	return starlark.NewBuiltin("home_dir", func(_ *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if len(kwargs) > 0 {
			return nil, fmt.Errorf("%s: unexpected keyword argument %s", b.Name(), kwargs[0][0])
		}
		parts := make([]string, len(args))
		for i, arg := range args {
			s, ok := starlark.AsString(arg)
			if !ok {
				return nil, fmt.Errorf("%s: argument %d: got %s, want string", b.Name(), i+1, arg.Type())
			}
			parts[i] = s
		}
		home, err := homedir.Expand("~")
		if err != nil {
			return nil, fmt.Errorf("%s: %w", b.Name(), err)
		}
		return starlark.String(filepath.Join(append([]string{home}, parts...)...)), nil
	})
}
