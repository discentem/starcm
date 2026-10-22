// Package extract implements extract(), which unpacks a .zip, .tar.gz or
// .tgz archive into a directory.
//
// Every entry is validated before anything is written, and files are written
// through an os.Root on the destination, so neither a hostile archive nor a
// symlink already inside the destination can make extract write outside it.
package extract

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/discentem/starcm/functions/base"
	"go.starlark.net/starlark"
)

type extractAction struct {
	goos string
	// checkStored verifies path if it names a file in the local store.
	// Nil means no store.
	checkStored func(path string) error
}

var _ base.Action = (*extractAction)(nil)

const (
	formatZip   = "zip"
	formatTarGz = "tar.gz"
)

// formatOf picks the archive format from the file name's extension.
func formatOf(path string) (string, error) {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return formatZip, nil
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return formatTarGz, nil
	}
	return "", fmt.Errorf("%s: unknown archive format; extract() handles .zip, .tar.gz and .tgz", path)
}

// entry is a regular file or directory in an archive.
type entry struct {
	// raw is the name as the archive spells it, which members matches.
	raw string
	// name is raw cleaned: slash separated, relative, without "." or ".."
	// elements. It's empty for the archive's root directory ("./").
	name  string
	isDir bool
	perm  os.FileMode
}

// cleanName checks that an archive entry name stays inside the destination
// and returns it slash separated.
func cleanName(raw string) (string, error) {
	if raw == "" {
		return "", errors.New("empty name")
	}
	if raw[0] == '/' || raw[0] == '\\' {
		return "", errors.New("absolute path")
	}
	if len(raw) >= 2 && raw[1] == ':' {
		return "", errors.New("absolute path")
	}
	var parts []string
	for _, p := range strings.FieldsFunc(raw, func(r rune) bool { return r == '/' || r == '\\' }) {
		switch p {
		case "..":
			return "", errors.New("path contains \"..\"")
		case ".":
		default:
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "/"), nil
}

// newEntry validates one archive entry. mode carries the entry's type bits.
func newEntry(raw string, mode fs.FileMode, perm os.FileMode) (entry, error) {
	name, err := cleanName(raw)
	if err != nil {
		return entry{}, fmt.Errorf("refusing entry %q: %v", raw, err)
	}
	switch {
	case mode&fs.ModeSymlink != 0:
		return entry{}, fmt.Errorf("refusing entry %q: symbolic links are not extracted", raw)
	case mode.IsDir():
		return entry{raw: raw, name: name, isDir: true}, nil
	case mode.IsRegular():
		if name == "" {
			return entry{}, fmt.Errorf("refusing entry %q: no file name", raw)
		}
		if perm &= 0o777; perm == 0 {
			perm = 0o644
		}
		return entry{raw: raw, name: name, perm: perm}, nil
	}
	return entry{}, fmt.Errorf("refusing entry %q: not a regular file or directory", raw)
}

// walk calls fn for each file and directory entry of the archive, in order.
// open returns the entry's content; it's only valid during fn. An entry that
// isn't safe to extract stops the walk with an error.
func walk(path, format string, fn func(e entry, open func() (io.Reader, error)) error) error {
	if format == formatZip {
		zr, err := zip.OpenReader(path)
		if err != nil {
			return err
		}
		defer zr.Close()
		for _, f := range zr.File {
			e, err := newEntry(f.Name, f.Mode(), f.Mode().Perm())
			if err != nil {
				return err
			}
			var rc io.ReadCloser
			err = fn(e, func() (io.Reader, error) {
				var err error
				rc, err = f.Open()
				return rc, err
			})
			if rc != nil {
				rc.Close()
			}
			if err != nil {
				return err
			}
		}
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		var mode fs.FileMode
		switch hdr.Typeflag {
		case tar.TypeXGlobalHeader:
			continue
		case tar.TypeReg, '\x00':
			mode = 0
		case tar.TypeDir:
			mode = fs.ModeDir
		case tar.TypeSymlink:
			mode = fs.ModeSymlink
		default: // hard links, devices, fifos, sparse files
			mode = fs.ModeIrregular
		}
		e, err := newEntry(hdr.Name, mode, os.FileMode(hdr.Mode))
		if err != nil {
			return err
		}
		if err := fn(e, func() (io.Reader, error) { return tr, nil }); err != nil {
			return err
		}
	}
}

func (a *extractAction) Execute(inst *base.FunctionInstance) (*base.Result, error) {
	ctx, label := inst.Ctx, inst.Label
	fail := func(format string, args ...any) error {
		return fmt.Errorf("extract(label=%q): %s", label, fmt.Sprintf(format, args...))
	}

	archive, _, err := inst.AbsPath("archive")
	if err != nil {
		return nil, err
	}
	dest, _, err := inst.AbsPath("dest")
	if err != nil {
		return nil, err
	}
	var members map[string]bool
	var memberOrder []string
	if list, ok := base.Get[*starlark.List](inst, "members"); ok {
		members = map[string]bool{}
		for i := 0; i < list.Len(); i++ {
			s, ok := starlark.AsString(list.Index(i))
			if !ok {
				return nil, fail("members must be a list of strings, not a list containing %s", list.Index(i).Type())
			}
			members[s] = true
			memberOrder = append(memberOrder, s)
		}
	}

	format, err := formatOf(archive)
	if err != nil {
		return nil, fail("%v", err)
	}
	if a.checkStored != nil {
		if err := a.checkStored(archive); err != nil {
			return nil, fail("%v", err)
		}
	}

	// Validate every entry before writing anything.
	var files []entry
	err = walk(archive, format, func(e entry, _ func() (io.Reader, error)) error {
		if !e.isDir {
			files = append(files, e)
		}
		return nil
	})
	if err != nil {
		return nil, fail("%v", err)
	}
	have := map[string]bool{}
	for _, e := range files {
		have[e.raw] = true
	}
	for _, m := range memberOrder {
		if !have[m] {
			return nil, fail("member %q not found in %s", m, archive)
		}
	}
	selected := func(e entry) bool { return members == nil || members[e.raw] }
	total, count := len(files), 0
	for _, e := range files {
		if selected(e) {
			count++
		}
	}

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, fail("%v", err)
	}
	root, err := os.OpenRoot(dest)
	if err != nil {
		return nil, fail("%v", err)
	}
	defer root.Close()
	tmp, err := os.MkdirTemp("", "starcm-extract-")
	if err != nil {
		return nil, fail("%v", err)
	}
	defer os.RemoveAll(tmp)

	w := &writer{root: root, tmp: tmp, goos: a.goos, dirs: map[string]bool{}}
	var paths []starlark.Value
	err = walk(archive, format, func(e entry, open func() (io.Reader, error)) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if e.isDir {
			// With members, only the directories files need are created.
			if members != nil || e.name == "" {
				return nil
			}
			return w.mkdirAll(e.name)
		}
		if !selected(e) {
			return nil
		}
		r, err := open()
		if err != nil {
			return fmt.Errorf("%s: %w", e.raw, err)
		}
		if err := w.writeFile(e, r); err != nil {
			return fmt.Errorf("%s: %w", e.raw, err)
		}
		paths = append(paths, starlark.String(filepath.Join(dest, filepath.FromSlash(e.name))))
		return nil
	})
	if err != nil {
		return nil, fail("%v", err)
	}

	msg := fmt.Sprintf("%q already up to date", dest)
	if w.changed {
		msg = fmt.Sprintf("extracted %d of %d files to %q", count, total, dest)
	}
	return &base.Result{
		Label:   label,
		Success: true,
		Changed: w.changed,
		Message: &msg,
		Return:  starlark.NewList(paths),
	}, nil
}

// writer puts files into root.
type writer struct {
	root    *os.Root
	tmp     string
	goos    string
	dirs    map[string]bool
	changed bool
}

// mkdirAll creates dir (slash separated, relative to root) and its parents.
func (w *writer) mkdirAll(dir string) error {
	cur := ""
	for _, p := range strings.Split(dir, "/") {
		if cur != "" {
			cur += "/"
		}
		cur += p
		if w.dirs[cur] {
			continue
		}
		info, err := w.root.Stat(cur)
		switch {
		case err == nil && !info.IsDir():
			return fmt.Errorf("%q exists and is not a directory", cur)
		case errors.Is(err, fs.ErrNotExist):
			if err := w.root.Mkdir(cur, 0o755); err != nil {
				return err
			}
			w.changed = true
		case err != nil:
			return err
		}
		w.dirs[cur] = true
	}
	return nil
}

// writeFile makes the file e.name in root hold r's content with e.perm. r is
// streamed to a private temp file while it's hashed, and copied into place
// only if the file there differs.
func (w *writer) writeFile(e entry, r io.Reader) error {
	if dir := pathDir(e.name); dir != "" {
		if err := w.mkdirAll(dir); err != nil {
			return err
		}
	}
	tmp, err := os.CreateTemp(w.tmp, "entry-")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, h), r); err != nil {
		return err
	}
	sum := hex.EncodeToString(h.Sum(nil))

	info, err := w.root.Lstat(e.name)
	switch {
	case err == nil:
		if info.IsDir() {
			return fmt.Errorf("%q is a directory, not a file", e.name)
		}
		// Windows only keeps a read-only bit, so perm can't be compared there.
		samePerm := w.goos == "windows" || info.Mode().Perm() == e.perm
		if info.Mode().IsRegular() && samePerm {
			if have, err := w.hash(e.name); err == nil && have == sum {
				return nil
			}
		}
		// Remove rather than overwrite: it also replaces a symlink with the
		// file instead of following it, and a read-only file.
		if err := w.root.Remove(e.name); err != nil {
			return err
		}
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}

	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return err
	}
	out, err := w.root.OpenFile(e.name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, e.perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, tmp); err != nil {
		out.Close()
		return err
	}
	// The mode OpenFile applied was reduced by the umask.
	if err := out.Chmod(e.perm); err != nil && w.goos != "windows" {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	w.changed = true
	return nil
}

func (w *writer) hash(name string) (string, error) {
	f, err := w.root.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func pathDir(name string) string {
	i := strings.LastIndexByte(name, '/')
	if i < 0 {
		return ""
	}
	return name[:i]
}

// extractArgs are extract()'s own arguments.
var extractArgs = []base.ArgPair{
	base.Arg[string]("archive"),
	base.Arg[string]("dest"),
	base.OptionalArgAllowNone[*starlark.List]("members"),
}

// Option configures extract().
type Option func(*extractAction)

// WithStoreCheck makes extract() call check on the archive path first, so an
// archive from the local store is re-hashed before it's read.
func WithStoreCheck(check func(path string) error) Option {
	return func(a *extractAction) { a.checkStored = check }
}

// New returns the extract() builtin.
//
//	extract(zip.value, dest = "/usr/local/lib/terraform", members = ["terraform"], label = "unpack terraform")
func New(ctx context.Context, opts ...Option) *base.Module {
	a := &extractAction{goos: runtime.GOOS}
	for _, opt := range opts {
		opt(a)
	}
	return base.NewModule(
		ctx,
		"extract",
		extractArgs,
		a,
	)
}
