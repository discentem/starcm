package template

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/discentem/starcm/functions/base"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/discentem/starcm/testhelpers/aferohelpers"
	"github.com/noirbizarre/gonja"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

// FileDefinition represents a file to be created in the test filesystem
type FileDefinition = aferohelpers.FileDefinition

// testRoot is an absolute directory for the test filesystem, so that
// destinations can be absolute on every platform.
func testRoot() string {
	if runtime.GOOS == "windows" {
		return `C:\repo`
	}
	return "/repo"
}

// p returns name under testRoot().
func p(name string) string {
	return filepath.Join(testRoot(), filepath.FromSlash(name))
}

// capture wraps an action to keep the Go Result it returns.
type capture struct {
	inner  base.Action
	result *base.Result
}

func (c *capture) Execute(inst *base.FunctionInstance) (*base.Result, error) {
	r, err := c.inner.Execute(inst)
	c.result = r
	return r, err
}

// runTemplate calls template() from a .star file in testRoot(), as a script
// would, and returns the Result the action produced.
func runTemplate(t *testing.T, fs afero.Fs, kwargs []starlark.Tuple) (*base.Result, error) {
	t.Helper()
	c := &capture{inner: &templateAction{fsys: fs}}
	fn := starlark.NewBuiltin("template", base.NewModule(context.Background(), "template", templateArgs, c).Function())
	thread := &starlark.Thread{Name: "test"}
	globals, err := starlark.ExecFile(thread, p("test.star"), "def call(**kw):\n    return template(label = \"template_test\", **kw)\n", starlark.StringDict{"template": fn})
	require.NoError(t, err)
	_, err = starlark.Call(thread, globals["call"], nil, kwargs)
	return c.result, err
}

func TestTemplateAction_Arguments(t *testing.T) {
	data := func() starlark.Value { return starlarkhelpers.GoDictToStarlarkDict(map[string]any{"key": "value"}) }
	kw := func(k string, v starlark.Value) starlark.Tuple { return starlark.Tuple{starlark.String(k), v} }
	tmpl := kw("template", starlark.String("path/to/template.tmpl"))
	dest := kw("destination", starlark.String(p("out.txt")))
	tests := []struct {
		name        string
		kwargs      []starlark.Tuple
		errContains string
	}{
		{"valid args", []starlark.Tuple{tmpl, kw("data", data()), dest}, ""},
		{"missing template", []starlark.Tuple{kw("data", data()), dest}, "needs template"},
		{"missing data", []starlark.Tuple{tmpl, dest}, "missing argument for data"},
		{"missing destination", []starlark.Tuple{tmpl, kw("data", data())}, "destination is required"},
		{"relative destination", []starlark.Tuple{tmpl, kw("data", data()), kw("destination", starlark.String("out.txt"))}, `"out.txt" is not an absolute path`},
		{"template and source", []starlark.Tuple{tmpl, kw("source", starlark.String("x")), kw("data", data()), dest}, "not both"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := aferohelpers.NewMemFsWithFiles(FileDefinition{Path: p("path/to/template.tmpl"), Content: "{{ key }}"})
			_, err := runTemplate(t, fs, tt.kwargs)
			if tt.errContains == "" {
				require.NoError(t, err)
				// A relative template resolves next to the calling .star file.
				content, err := afero.ReadFile(fs, p("out.txt"))
				require.NoError(t, err)
				require.Equal(t, "value", string(content))
				return
			}
			require.ErrorContains(t, err, tt.errContains)
		})
	}
}

func TestTemplateAction_writeTemplate(t *testing.T) {
	tests := []struct {
		name            string
		destinationPath string
		finalContent    []byte
		setupFs         func() afero.Fs
		wantErr         bool
		assertFs        func(t *testing.T, fs afero.Fs, path string, content []byte)
	}{
		{
			name:            "write to new file",
			destinationPath: "output.txt",
			finalContent:    []byte("rendered content"),
			setupFs:         func() afero.Fs { return afero.NewMemMapFs() },
			wantErr:         false,
			assertFs: func(t *testing.T, fs afero.Fs, path string, content []byte) {
				exists, err := afero.Exists(fs, path)
				assert.NoError(t, err)
				assert.True(t, exists)

				data, err := afero.ReadFile(fs, path)
				assert.NoError(t, err)
				assert.Equal(t, content, data)
			},
		},
		{
			name:            "write to nested directory",
			destinationPath: "path/to/nested/output.txt",
			finalContent:    []byte("rendered content"),
			setupFs:         func() afero.Fs { return afero.NewMemMapFs() },
			wantErr:         false,
			assertFs: func(t *testing.T, fs afero.Fs, path string, content []byte) {
				exists, err := afero.Exists(fs, path)
				assert.NoError(t, err)
				assert.True(t, exists)

				data, err := afero.ReadFile(fs, path)
				assert.NoError(t, err)
				assert.Equal(t, content, data)
			},
		},
		{
			name:            "overwrite existing file",
			destinationPath: "existing.txt",
			finalContent:    []byte("new content"),
			setupFs: func() afero.Fs {
				return aferohelpers.NewMemFsWithFiles(
					FileDefinition{
						Path:    "existing.txt",
						Content: "old content",
					},
				)
			},
			wantErr: false,
			assertFs: func(t *testing.T, fs afero.Fs, path string, content []byte) {
				exists, err := afero.Exists(fs, path)
				assert.NoError(t, err)
				assert.True(t, exists)

				data, err := afero.ReadFile(fs, path)
				assert.NoError(t, err)
				assert.Equal(t, content, data)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := tt.setupFs()
			action := &templateAction{fsys: fs}
			err := action.writeTemplate(
				tt.destinationPath,
				tt.finalContent,
				writeTemplateOptions{
					persist: true,
				})

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			tt.assertFs(t, fs, tt.destinationPath, tt.finalContent)
		})
	}
}

func TestTemplateAction_Run(t *testing.T) {
	tests := []struct {
		name              string
		setupFs           func() (afero.Fs, string)
		args              starlark.Tuple
		kwargs            []starlark.Tuple
		expectedSuccess   bool
		expectedChanged   bool
		expectedOutput    string
		expectedDiff      string
		expectOutputEqual bool
		wantErr           bool
	}{
		{
			name: "successful template rendering - new file",
			setupFs: func() (afero.Fs, string) {
				workDir := ""
				fs := aferohelpers.NewMemFsWithFiles(
					FileDefinition{
						Path:    p("template.tmpl"),
						Content: "Hello {{ name }}!",
					},
				)
				return fs, workDir
			},
			kwargs: []starlark.Tuple{
				{starlark.String("template"), starlark.String("template.tmpl")},
				{starlark.String("data"), starlarkhelpers.GoDictToStarlarkDict(map[string]any{"name": "World"})},
				{starlark.String("destination"), starlark.String(p("output.txt"))},
			},
			expectedSuccess:   true,
			expectedChanged:   true,
			expectedOutput:    "Hello World!",
			expectOutputEqual: true,
			wantErr:           false,
		},
		{
			name: "no change when content is the same",
			setupFs: func() (afero.Fs, string) {
				workDir := ""
				fs := aferohelpers.NewMemFsWithFiles(
					FileDefinition{
						Path:    p("template.tmpl"),
						Content: "Hello {{ name }}!",
					},
					FileDefinition{
						Path:    p("output.txt"),
						Content: "Hello World!",
					},
				)
				return fs, workDir
			},
			kwargs: []starlark.Tuple{
				{starlark.String("template"), starlark.String("template.tmpl")},
				{starlark.String("data"), starlarkhelpers.GoDictToStarlarkDict(map[string]any{"name": "World"})},
				{starlark.String("destination"), starlark.String(p("output.txt"))},
			},
			expectedSuccess:   true,
			expectedChanged:   false,
			expectedOutput:    "Hello World!",
			expectedDiff:      "",
			expectOutputEqual: true,
			wantErr:           false,
		},
		{
			name: "error when template file does not exist",
			setupFs: func() (afero.Fs, string) {
				return afero.NewMemMapFs(), ""
			},
			kwargs: []starlark.Tuple{
				{starlark.String("template"), starlark.String("nonexistent.tmpl")},
				{starlark.String("data"), starlarkhelpers.GoDictToStarlarkDict(map[string]any{"name": "World"})},
				{starlark.String("destination"), starlark.String(p("output.txt"))},
			},
			expectedSuccess: false,
			expectedChanged: false,
			wantErr:         true,
		},
		{
			name: "content changed - existing file with different content",
			setupFs: func() (afero.Fs, string) {
				fs := aferohelpers.NewMemFsWithFiles(
					FileDefinition{
						Path:    p("template.tmpl"),
						Content: "Hello {{ name }}!",
					},
					FileDefinition{
						Path:    p("output.txt"),
						Content: "Different content",
					},
				)
				return fs, ""
			},
			kwargs: []starlark.Tuple{
				{starlark.String("template"), starlark.String("template.tmpl")},
				{starlark.String("data"), starlarkhelpers.GoDictToStarlarkDict(map[string]any{"name": "World"})},
				{starlark.String("destination"), starlark.String(p("output.txt"))},
			},
			expectedSuccess:   true,
			expectedChanged:   true,
			expectedOutput:    "Hello World!",
			expectedDiff:      "  []string{\n- \t\"Different content\",\n+ \t\"Hello World!\",\n  }\n",
			expectOutputEqual: true,
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, _ := tt.setupFs()
			result, err := runTemplate(t, fs, tt.kwargs)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, tt.expectedSuccess, result.Success)
			require.Equal(t, tt.expectedChanged, result.Changed)

			if tt.expectOutputEqual && result.Message != nil {
				require.Equal(t, tt.expectedOutput, *result.Message)
			}

			if tt.expectedDiff != "" && result.Diff != nil {
				require.Equal(t, tt.expectedDiff, *result.Diff)
			}

			// Verify destination file content if it was successful
			if result.Success {
				destinationPath := ""
				for _, kw := range tt.kwargs {
					if kw[0].(starlark.String) == "destination" {
						destinationPath = string(kw[1].(starlark.String))
						break
					}
				}
				if destinationPath != "" {
					content, err := afero.ReadFile(fs, destinationPath)
					require.NoError(t, err)
					require.Equal(t, tt.expectedOutput, string(content))
				}
			}
		})
	}
}

// Test that template errors are properly handled
func TestTemplateAction_TemplateErrors(t *testing.T) {
	tests := []struct {
		name            string
		templateContent string
		data            map[string]any
		wantErr         bool
		errContains     string
	}{
		{
			name:            "invalid template syntax",
			templateContent: "Hello {{ name }", // Missing closing brace
			data:            map[string]any{"name": "World"},
			wantErr:         true,
		},
		{
			name:            "undefined variable",
			templateContent: "Hello {{ undefined_var }}!",
			data:            map[string]any{"name": "World"},
			wantErr:         false, // gonja doesn't error on undefined vars, just renders empty
		},
		{
			name:            "valid complex template",
			templateContent: "{% if name %}Hello {{ name }}!{% else %}Hello anonymous!{% endif %}",
			data:            map[string]any{"name": "World"},
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := aferohelpers.NewMemFsWithFiles(
				FileDefinition{
					Path:    p("template.tmpl"),
					Content: tt.templateContent,
				},
			)

			kwargs := []starlark.Tuple{
				{starlark.String("template"), starlark.String("template.tmpl")},
				{starlark.String("data"), starlarkhelpers.GoDictToStarlarkDict(tt.data)},
				{starlark.String("destination"), starlark.String(p("output.txt"))},
			}

			result, err := runTemplate(t, fs, kwargs)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.True(t, result.Success)

			// Verify file was created with correct content
			content, err := afero.ReadFile(fs, p("output.txt"))
			assert.NoError(t, err)

			// Using gonja directly to verify expected output
			tmpl, err := gonja.FromString(tt.templateContent)
			assert.NoError(t, err)
			expected, err := tmpl.Execute(tt.data)
			assert.NoError(t, err)

			assert.Equal(t, expected, string(content))
		})
	}
}

// Test edge cases for the file system operations
func TestTemplateAction_FileSystemEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupFs     func() afero.Fs
		destination string
		wantErr     bool
	}{
		{
			name: "destination directory already exists",
			setupFs: func() afero.Fs {
				return aferohelpers.NewMemFsWithFiles(
					FileDefinition{
						Path:    p("template.tmpl"),
						Content: "content",
					},
					FileDefinition{
						Path:  p("output/dir"),
						IsDir: true,
					},
				)
			},
			destination: "output/dir/file.txt",
			wantErr:     false,
		},
		{
			name: "destination is a directory",
			setupFs: func() afero.Fs {
				return aferohelpers.NewMemFsWithFiles(
					FileDefinition{
						Path:    p("template.tmpl"),
						Content: "content",
					},
					FileDefinition{
						Path:  p("output"),
						IsDir: true,
					},
				)
			},
			destination: "output",
			wantErr:     true, // Should error since destination is a directory
		},
		{
			name: "custom modification time preservation",
			setupFs: func() afero.Fs {
				pastTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
				return aferohelpers.NewMemFsWithFiles(
					FileDefinition{
						Path:    p("template.tmpl"),
						Content: "content",
						ModTime: pastTime,
					},
				)
			},
			destination: "output.txt",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := tt.setupFs()

			kwargs := []starlark.Tuple{
				{starlark.String("template"), starlark.String("template.tmpl")},
				{starlark.String("data"), starlarkhelpers.GoDictToStarlarkDict(map[string]any{"key": "value"})},
				{starlark.String("destination"), starlark.String(p(tt.destination))},
			}

			result, err := runTemplate(t, fs, kwargs)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.True(t, result.Success)

			// Verify destination file exists
			exists, err := afero.Exists(fs, p(tt.destination))
			assert.NoError(t, err)
			assert.True(t, exists)
		})
	}
}
