// Package argcompat_test checks that every starcm function behaves the same
// whether its leading arguments are passed positionally or by keyword.
// base parses each call once into a FunctionInstance, so the two forms must
// never diverge; a case that does is a bug in base or in that function's
// argument declarations.
package argcompat_test

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/discentem/starcm/functions/base"
	starcmchecksum "github.com/discentem/starcm/functions/checksum"
	starcmdate "github.com/discentem/starcm/functions/date"
	loading "github.com/discentem/starcm/functions/dynamic_loading"
	starcmextract "github.com/discentem/starcm/functions/extract"
	starcmfile "github.com/discentem/starcm/functions/file"
	starcmhere "github.com/discentem/starcm/functions/here"
	starcminspect "github.com/discentem/starcm/functions/inspect"
	starcmpathmod "github.com/discentem/starcm/functions/pathmod"
	starcmreadfile "github.com/discentem/starcm/functions/readfile"
	starcmshell "github.com/discentem/starcm/functions/shell"
	starcmtemplate "github.com/discentem/starcm/functions/template"
	starcmwrite "github.com/discentem/starcm/functions/write"
	shelllib "github.com/discentem/starcm/libraries/shell"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

type argCase struct {
	name    string
	keyword string // call using keyword arguments only
	pos     string // the same call with leading arguments passed positionally
}

// echo is a shell builtin on Windows, so run it through cmd there.
var echoCmd, echoArgs = func() (string, string) {
	if runtime.GOOS == "windows" {
		return `"cmd"`, `["/c", "echo hi"]`
	}
	return `"echo"`, `["hi"]`
}()

var cases = []argCase{
	{"read_file", `read_file(label = "x", path = here("in.txt"))`, `read_file(here("in.txt"), label = "x")`},
	{"sha256", `sha256(label = "x", path = here("in.txt"))`, `sha256(here("in.txt"), label = "x")`},
	{"inspect", `inspect(label = "x", path = here("mod.star"))`, `inspect(here("mod.star"), label = "x")`},
	{"path.list/1-positional", `path.list(dir = here("listdir"), pattern = "*.txt")`, `path.list(here("listdir"), pattern = "*.txt")`},
	{"path.list/2-positional", `path.list(dir = here("listdir"), pattern = "*.txt")`, `path.list(here("listdir"), "*.txt")`},
	{"path.exists", `path.exists(path = here("in.txt"))`, `path.exists(here("in.txt"))`},
	{"path.glob", `path.glob(pattern = here("listdir/*.txt"))`, `path.glob(here("listdir/*.txt"))`},
	{"write", `write(label = "x", str = "hello")`, `write("hello", label = "x")`},
	{"file", `file(label = "x", path = here("out.txt"), content = "hi")`, `file(here("out.txt"), content = "hi", label = "x")`},
	{"file/copy", `file(label = "x", path = here("copy.txt"), source = here("in.txt"))`, `file(here("copy.txt"), source = here("in.txt"), label = "x")`},
	{"extract/1-positional", `extract(label = "x", archive = here("a.zip"), dest = here("out"))`, `extract(here("a.zip"), dest = here("out"), label = "x")`},
	{"extract/2-positional", `extract(label = "x", archive = here("a.zip"), dest = here("out"))`, `extract(here("a.zip"), here("out"), label = "x")`},
	{"exec/1-positional", `exec(label = "x", cmd = ` + echoCmd + `, args = ` + echoArgs + `)`, `exec(` + echoCmd + `, args = ` + echoArgs + `, label = "x")`},
	{"exec/2-positional", `exec(label = "x", cmd = ` + echoCmd + `, args = ` + echoArgs + `)`, `exec(` + echoCmd + `, ` + echoArgs + `, label = "x")`},
	{"template", `template(label = "x", template = "t.tmpl", data = {"name": "a"}, destination = here("t.txt"))`, `template("t.tmpl", data = {"name": "a"}, destination = here("t.txt"), label = "x")`},
	{"load_dynamic", `load_dynamic(label = "x", module = "mod.star").value["x"]`, `load_dynamic("mod.star", label = "x").value["x"]`},
	// date returns the time of the call; compare only the part format controls.
	{"date", `date(label = "x", format = "2006").value.formatted`, `date("2006", label = "x").value.formatted`},
}

// fixtures creates the files the cases read, inside dir.
func fixtures(t *testing.T, dir string) {
	t.Helper()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.WriteFile(filepath.Join(dir, "in.txt"), []byte("hello\n"), 0o644))
	must(os.WriteFile(filepath.Join(dir, "mod.star"), []byte("x = 1\n"), 0o644))
	must(os.WriteFile(filepath.Join(dir, "t.tmpl"), []byte("hi {{ name }}\n"), 0o644))
	must(os.MkdirAll(filepath.Join(dir, "listdir"), 0o755))
	must(os.WriteFile(filepath.Join(dir, "listdir", "a.txt"), []byte("a"), 0o644))

	f, err := os.Create(filepath.Join(dir, "a.zip"))
	must(err)
	zw := zip.NewWriter(f)
	w, err := zw.Create("inner.txt")
	must(err)
	_, err = w.Write([]byte("inner"))
	must(err)
	must(zw.Close())
	must(f.Close())
}

// predeclared builds the functions under test the way libraries/loader does,
// minus network, store and keyring options.
func predeclared(ctx context.Context, out *bytes.Buffer) starlark.StringDict {
	fsys := afero.NewOsFs()
	mods := map[string]*base.Module{
		"read_file":    starcmreadfile.New(ctx, fsys),
		"sha256":       starcmchecksum.New(ctx, fsys),
		"inspect":      starcminspect.New(ctx, fsys),
		"write":        starcmwrite.New(ctx, out),
		"file":         starcmfile.New(ctx, fsys),
		"extract":      starcmextract.New(ctx),
		"exec":         starcmshell.New(ctx, &shelllib.RealExecutor{}),
		"template":     starcmtemplate.New(ctx, fsys),
		"load_dynamic": loading.New(ctx),
		"date":         starcmdate.New(ctx),
	}
	d := starlark.StringDict{
		"here": starcmhere.New(func(string) bool { return false }),
		"path": starcmpathmod.New(ctx, fsys),
	}
	for name, m := range mods {
		d[name] = starlark.NewBuiltin(name, m.Function())
	}
	return d
}

// load runs a .star file by its absolute path, enough for load_dynamic.
func load(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	return starlark.ExecFile(&starlark.Thread{Name: module}, filepath.FromSlash(module), nil, nil)
}

// run executes expr in a fresh copy of the fixtures and describes the outcome.
func run(t *testing.T, expr string) string {
	t.Helper()
	dir := t.TempDir()
	fixtures(t, dir)
	var out bytes.Buffer
	thread := &starlark.Thread{Name: "argcompat", Load: load}
	src := "r = " + expr + "\n"
	globals, err := starlark.ExecFile(thread, filepath.Join(dir, "case.star"), src, predeclared(context.Background(), &out))
	// Values print with Starlark string escapes, so on Windows a path's
	// backslashes appear doubled, and some functions return
	// paths with forward slashes; mask every form of the temp dir.
	norm := func(s string) string {
		dirs := []string{dir}
		if real, err := filepath.EvalSymlinks(dir); err == nil {
			dirs = append(dirs, real)
		}
		for _, d := range dirs {
			s = strings.ReplaceAll(s, strings.ReplaceAll(d, `\`, `\\`), "$DIR")
			s = strings.ReplaceAll(s, filepath.ToSlash(d), "$DIR")
			s = strings.ReplaceAll(s, d, "$DIR")
		}
		return s
	}
	if err != nil {
		return norm("call error: " + firstLine(err.Error()))
	}
	r, ok := globals["r"].(*starlarkstruct.Struct)
	if !ok {
		return norm(fmt.Sprintf("value=%s", globals["r"]))
	}
	field := func(name string) string {
		v, err := r.Attr(name)
		if err != nil {
			return "<missing>"
		}
		return v.String()
	}
	return norm(fmt.Sprintf("success=%s value=%s error=%s output=%q",
		field("success"), field("value"), firstLine(field("error")), out.String()))
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func TestPositionalMatchesKeyword(t *testing.T) {
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kw, pos := run(t, c.keyword), run(t, c.pos)
			if strings.HasPrefix(kw, "call error") || strings.Contains(kw, "success=False") {
				t.Errorf("the keyword form must succeed, so the comparison means something\n  keyword: %s", kw)
			}
			if kw != pos {
				t.Errorf("positional differs from keyword\n  keyword:    %s\n  positional: %s", kw, pos)
			}
			if testing.Verbose() {
				t.Logf("both: %s", kw)
			}
		})
	}
}
