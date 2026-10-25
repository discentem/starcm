// Package glob implements glob(), which finds the paths that match a pattern
// whose wildcards can appear in any path segment, directories included.
package glob

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/discentem/starcm/functions/base"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

// Glob returns the paths that match pattern, an absolute path whose segments
// may hold the wildcards of filepath.Match: *, ? and [...]. Each wildcard
// matches within one segment. Files and directories both match. The paths
// are sorted and use the platform's separator.
//
// Unlike filepath.Glob, an error reading the filesystem (permission denied,
// say) is returned rather than skipped, so a directory starcm can't read
// never looks empty. A segment that doesn't exist, or a file where the
// pattern needs a directory, just matches nothing.
func Glob(fsys afero.Fs, pattern string) ([]string, error) {
	if err := validate(pattern); err != nil {
		return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
	}
	if !filepath.IsAbs(pattern) {
		return nil, fmt.Errorf("pattern %q is not absolute", pattern)
	}
	pattern = filepath.Clean(pattern)
	vol := filepath.VolumeName(pattern)
	// Clean leaves exactly one separator after the volume of an absolute path.
	root := pattern[:len(vol)+1]
	rest := pattern[len(root):]
	if rest == "" {
		return []string{root}, nil
	}
	matches := []string{root}
	segments := strings.Split(rest, string(filepath.Separator))
	for i, seg := range segments {
		last := i == len(segments)-1
		var next []string
		for _, dir := range matches {
			found, err := matchSegment(fsys, dir, seg, last)
			if err != nil {
				return nil, err
			}
			next = append(next, found...)
		}
		matches = next
		if len(matches) == 0 {
			break
		}
	}
	sort.Strings(matches)
	return matches, nil
}

// validate checks every part of pattern. filepath.Match(pattern, "") alone
// misses errors: it stops at the first part that doesn't match the empty
// name, so "a*[" would pass. Match does check the whole of a pattern without
// a *, so each run of text between stars is checked on its own.
func validate(pattern string) error {
	for _, chunk := range chunks(pattern) {
		if _, err := filepath.Match(chunk, ""); err != nil {
			return err
		}
	}
	return nil
}

// chunks splits pattern at each * that isn't inside [...], dropping the
// stars, the way filepath.Match scans a pattern.
func chunks(pattern string) []string {
	var out []string
	start, inrange := 0, false
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '\\':
			if runtime.GOOS != "windows" && i+1 < len(pattern) {
				i++
			}
		case '[':
			inrange = true
		case ']':
			inrange = false
		case '*':
			if !inrange {
				out = append(out, pattern[start:i])
				start = i + 1
			}
		}
	}
	return append(out, pattern[start:])
}

// matchSegment returns the entries of dir that match seg. When seg isn't the
// last segment, only directories (or links to them) are kept, since the next
// segment looks inside them.
func matchSegment(fsys afero.Fs, dir, seg string, last bool) ([]string, error) {
	var candidates []string
	if hasMeta(seg) {
		entries, err := afero.ReadDir(fsys, dir)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			ok, err := filepath.Match(seg, e.Name())
			if err != nil {
				return nil, err
			}
			if ok {
				candidates = append(candidates, filepath.Join(dir, e.Name()))
			}
		}
	} else {
		candidates = []string{filepath.Join(dir, unescape(seg))}
	}
	var found []string
	for _, p := range candidates {
		var err error
		var info fs.FileInfo
		if last {
			info, err = lstat(fsys, p)
		} else {
			info, err = fsys.Stat(p)
		}
		switch {
		case errors.Is(err, fs.ErrNotExist):
			continue
		case err != nil:
			return nil, err
		case !last && !info.IsDir():
			continue
		}
		found = append(found, p)
	}
	return found, nil
}

func lstat(fsys afero.Fs, p string) (fs.FileInfo, error) {
	if l, ok := fsys.(afero.Lstater); ok {
		info, _, err := l.LstatIfPossible(p)
		return info, err
	}
	return fsys.Stat(p)
}

// hasMeta reports whether seg holds any of filepath.Match's special
// characters. \ escapes them everywhere but Windows, where it's the separator.
func hasMeta(seg string) bool {
	magic := `*?[\`
	if runtime.GOOS == "windows" {
		magic = `*?[`
	}
	return strings.ContainsAny(seg, magic)
}

// unescape removes the \ escapes from a segment without wildcards. On
// Windows there are none, since hasMeta treats every \ as a separator there.
func unescape(seg string) string {
	if runtime.GOOS == "windows" {
		return seg
	}
	var b strings.Builder
	for i := 0; i < len(seg); i++ {
		if seg[i] == '\\' && i+1 < len(seg) {
			i++
		}
		b.WriteByte(seg[i])
	}
	return b.String()
}

type globAction struct {
	fsys afero.Fs
}

var _ base.Action = (*globAction)(nil)

// Execute returns the paths matching the pattern argument as a sorted list
// of absolute paths with forward slashes on every platform. pattern must be
// absolute; a leading ~ is expanded.
func (a *globAction) Execute(inst *base.FunctionInstance) (*base.Result, error) {
	pattern, _, err := inst.AbsPath("pattern")
	if err != nil {
		return nil, err
	}
	matches, err := Glob(a.fsys, pattern)
	if err != nil {
		return nil, fmt.Errorf("glob(label=%q): %w", inst.Label, err)
	}
	list := make([]starlark.Value, len(matches))
	for i, m := range matches {
		list[i] = starlark.String(filepath.ToSlash(m))
	}
	return &base.Result{Label: inst.Label, Success: true, Return: starlark.NewList(list)}, nil
}

// New returns the glob() builtin.
//
//	mods = glob(here("third_party/*/go.mod"), label = "find modules").value
func New(ctx context.Context, fsys afero.Fs) *base.Module {
	return base.NewModule(
		ctx,
		"glob",
		[]base.ArgPair{
			base.Arg[string]("pattern"),
		},
		&globAction{fsys: fsys},
	)
}
