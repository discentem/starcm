package file

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/libraries/fileutils"
	"github.com/discentem/starcm/libraries/macsig"
	starcmsha256 "github.com/discentem/starcm/libraries/sha256"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

// absPath expands a leading ~ and makes path absolute.
func absPath(path string) (string, error) {
	p, err := homedir.Expand(path)
	if err != nil {
		return "", fmt.Errorf("failed to expand home directory in path %q: %w", path, err)
	}
	if !filepath.IsAbs(p) {
		if p, err = filepath.Abs(p); err != nil {
			return "", fmt.Errorf("failed to resolve path %q: %w", path, err)
		}
	}
	return p, nil
}

// runCopy makes path a copy of source. The copy is written to a private
// directory next to path, checked there (sha256, then signature), and only
// then renamed into place, so what lands at path is exactly what was
// checked. It works on the real filesystem, since pkgutil and codesign need
// real files.
func (a *fileAction) runCopy(label string, kwargs []starlark.Tuple) (*base.Result, error) {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("file(label=%q): %s", label, fmt.Sprintf(format, args...))
	}
	if starlarkhelpers.FindOptionalValueInKwargs(kwargs, "content") != nil {
		return nil, fail("give content or source, not both")
	}
	path, err := starlarkhelpers.FindValueinKwargs(kwargs, "path")
	if err != nil {
		return nil, err
	}
	dst, err := absPath(*path)
	if err != nil {
		return nil, err
	}
	srcArg, err := starlarkhelpers.FindValueinKwargs(kwargs, "source")
	if err != nil {
		return nil, err
	}
	src, err := absPath(*srcArg)
	if err != nil {
		return nil, err
	}
	want, err := optionalString(kwargs, "sha256")
	if err != nil {
		return nil, err
	}
	if want = strings.ToLower(want); want != "" && !sha256Hex.MatchString(want) {
		return nil, fail("sha256 %q is not 64 hex characters", want)
	}
	signer, err := optionalString(kwargs, "signer")
	if err != nil {
		return nil, err
	}
	requireNotarized, err := starlarkhelpers.FindOptionalBoolInKwargs(kwargs, "require_notarized", true)
	if err != nil {
		return nil, err
	}
	createDirs, err := starlarkhelpers.FindBoolInKwargs(kwargs, "create_dirs", false)
	if err != nil {
		return nil, err
	}
	if signer != "" && a.goos != "darwin" {
		return nil, fail("signer only works on macOS")
	}

	if a.checkStored != nil {
		if err := a.checkStored(src); err != nil {
			return nil, fail("%v", err)
		}
	}
	srcInfo, err := os.Stat(src)
	if err != nil {
		return nil, fail("%v", err)
	}
	if !srcInfo.Mode().IsRegular() {
		return nil, fail("source %q is not a regular file", src)
	}
	// mode defaults to the source's permissions, so executables stay
	// executable.
	mode := srcInfo.Mode().Perm()
	if starlarkhelpers.FindOptionalValueInKwargs(kwargs, "mode") != nil {
		m, err := starlarkhelpers.FindIntInKwargs(kwargs, "mode", 0)
		if err != nil {
			return nil, err
		}
		mode = os.FileMode(m).Perm()
	}

	dir := filepath.Dir(dst)
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		if !createDirs {
			return nil, fmt.Errorf("directory %q does not exist and create_dirs is false", dir)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create directories %q: %w", dir, err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to stat directory %q: %w", dir, err)
	}

	// MkdirTemp creates the directory 0700, so only this user can touch the
	// copy. It's next to dst so the rename below stays on one filesystem.
	tmp, err := os.MkdirTemp(dir, ".starcm-file-")
	if err != nil {
		return nil, fail("%v", err)
	}
	defer os.RemoveAll(tmp)
	staged := filepath.Join(tmp, filepath.Base(dst))
	sum, err := fileutils.CopyNew(src, staged, 0o600)
	if err != nil {
		return nil, fail("%v", err)
	}
	if want != "" && sum != want {
		return nil, fail("%s has sha256 %s, not the expected %s", src, sum, want)
	}

	var value starlark.Value = starlark.None
	if signer != "" {
		kind, err := macsig.ResolveKind(src, macsig.KindAuto)
		if err != nil {
			return nil, err
		}
		sig, err := macsig.New(a.executor).Read(staged, kind)
		if err != nil {
			return nil, fail("%s: %v", src, err)
		}
		if err := sig.Check(signer, requireNotarized); err != nil {
			return nil, fail("%s: %v", src, err)
		}
		fields := sig.Fields()
		fields["sha256"] = starlark.String(sum)
		fields["kind"] = starlark.String(kind)
		value = starlarkstruct.FromStringDict(starlark.String("mac_signature"), fields)
	}

	if info, err := os.Lstat(dst); err == nil {
		if info.IsDir() {
			return nil, fmt.Errorf("%q is a directory, not a file", dst)
		}
		// Windows only keeps a read-only bit, so mode can't be compared there.
		sameMode := a.goos == "windows" || info.Mode().Perm() == mode
		if info.Mode().IsRegular() && sameMode {
			if have, err := starcmsha256.File(afero.NewOsFs(), dst); err == nil && have == sum {
				msg := fmt.Sprintf("file %q already matches %q", dst, src)
				return &base.Result{Label: label, Success: true, Message: &msg, Return: value}, nil
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to stat file %q: %w", dst, err)
	}

	if err := os.Chmod(staged, mode); err != nil {
		return nil, fail("%v", err)
	}
	if err := os.Rename(staged, dst); err != nil {
		return nil, fail("%v", err)
	}
	msg := fmt.Sprintf("copied %q to %q", src, dst)
	return &base.Result{Label: label, Success: true, Changed: true, Message: &msg, Return: value}, nil
}

// optionalString is the string argument name, or "" when it's missing or
// None.
func optionalString(kwargs []starlark.Tuple, name string) (string, error) {
	v := starlarkhelpers.FindOptionalValueInKwargs(kwargs, name)
	if v == nil {
		return "", nil
	}
	s, ok := starlark.AsString(v)
	if !ok {
		return "", fmt.Errorf("%s must be a string, not %s", name, v.Type())
	}
	return strings.TrimSpace(s), nil
}
