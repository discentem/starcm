package template

import (
	"context"
	"path/filepath"
	"testing"
	"time"

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

func TestTemplateAction_parseArgs(t *testing.T) {
	tests := []struct {
		name     string
		kwargs   []starlark.Tuple
		wantErr  bool
		expected *parsedArgs
	}{
		{
			name: "valid args",
			kwargs: []starlark.Tuple{
				{starlark.String("template"), starlark.String("path/to/template.tmpl")},
				{starlark.String("data"), starlarkhelpers.GoDictToStarlarkDict(map[string]any{"key": "value"})},
				{starlark.String("destination"), starlark.String("path/to/output.txt")},
			},
			wantErr: false,
			expected: &parsedArgs{
				templatePath: "path/to/template.tmpl",
				data:         map[string]any{"key": "value"},
				destination:  "path/to/output.txt",
			},
		},
		{
			name: "missing template",
			kwargs: []starlark.Tuple{
				{starlark.String("data"), starlarkhelpers.GoDictToStarlarkDict(map[string]any{"key": "value"})},
				{starlark.String("destination"), starlark.String("path/to/output.txt")},
			},
			wantErr: true,
		},
		{
			name: "missing data",
			kwargs: []starlark.Tuple{
				{starlark.String("template"), starlark.String("path/to/template.tmpl")},
				{starlark.String("destination"), starlark.String("path/to/output.txt")},
			},
			wantErr: true,
		},
		{
			name: "missing destination",
			kwargs: []starlark.Tuple{
				{starlark.String("template"), starlark.String("path/to/template.tmpl")},
				{starlark.String("data"), starlarkhelpers.GoDictToStarlarkDict(map[string]any{"key": "value"})},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := &templateAction{fsys: afero.NewMemMapFs()}
			got, err := action.parseArgs(starlark.Tuple{}, tt.kwargs)

			if tt.wantErr {
				require.Error(t, err)
			} else {

				require.NoError(t, err)
				require.Equal(t, tt.expected.templatePath, got.templatePath)
				require.Equal(t, tt.expected.destination, got.destination)
				require.Equal(t, len(tt.expected.data), len(got.data))

				for k, v := range tt.expected.data {
					require.Equal(t, v, got.data[k])
				}
			}
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
			err := action.writeTemplate(tt.destinationPath, tt.finalContent)

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
						Path:    "template.tmpl",
						Content: "Hello {{ name }}!",
					},
				)
				return fs, workDir
			},
			kwargs: []starlark.Tuple{
				{starlark.String("template"), starlark.String("template.tmpl")},
				{starlark.String("data"), starlarkhelpers.GoDictToStarlarkDict(map[string]any{"name": "World"})},
				{starlark.String("destination"), starlark.String("output.txt")},
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
						Path:    "template.tmpl",
						Content: "Hello {{ name }}!",
					},
					FileDefinition{
						Path:    "output.txt",
						Content: "Hello World!",
					},
				)
				return fs, workDir
			},
			kwargs: []starlark.Tuple{
				{starlark.String("template"), starlark.String("template.tmpl")},
				{starlark.String("data"), starlarkhelpers.GoDictToStarlarkDict(map[string]any{"name": "World"})},
				{starlark.String("destination"), starlark.String("output.txt")},
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
				{starlark.String("destination"), starlark.String("output.txt")},
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
						Path:    "template.tmpl",
						Content: "Hello {{ name }}!",
					},
					FileDefinition{
						Path:    "output.txt",
						Content: "Different content",
					},
				)
				return fs, ""
			},
			kwargs: []starlark.Tuple{
				{starlark.String("template"), starlark.String("template.tmpl")},
				{starlark.String("data"), starlarkhelpers.GoDictToStarlarkDict(map[string]any{"name": "World"})},
				{starlark.String("destination"), starlark.String("output.txt")},
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
			fs, workDir := tt.setupFs()

			action := &templateAction{fsys: fs}
			ctx := context.Background()
			result, err := action.Run(ctx, workDir, "template_test", tt.args, tt.kwargs)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, tt.expectedSuccess, result.Success)
			require.Equal(t, tt.expectedChanged, result.Changed)

			if tt.expectOutputEqual && result.Output != nil {
				require.Equal(t, tt.expectedOutput, *result.Output)
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
					content, err := afero.ReadFile(fs, filepath.Join(workDir, destinationPath))
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
					Path:    "template.tmpl",
					Content: tt.templateContent,
				},
			)

			action := &templateAction{fsys: fs}
			ctx := context.Background()

			kwargs := []starlark.Tuple{
				{starlark.String("template"), starlark.String("template.tmpl")},
				{starlark.String("data"), starlarkhelpers.GoDictToStarlarkDict(tt.data)},
				{starlark.String("destination"), starlark.String("output.txt")},
			}

			result, err := action.Run(ctx, "", "template_test", starlark.Tuple{}, kwargs)

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
			content, err := afero.ReadFile(fs, "output.txt")
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
						Path:    "template.tmpl",
						Content: "content",
					},
					FileDefinition{
						Path:  "output/dir",
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
						Path:    "template.tmpl",
						Content: "content",
					},
					FileDefinition{
						Path:  "output",
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
						Path:    "template.tmpl",
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
			action := &templateAction{fsys: fs}
			ctx := context.Background()

			kwargs := []starlark.Tuple{
				{starlark.String("template"), starlark.String("template.tmpl")},
				{starlark.String("data"), starlarkhelpers.GoDictToStarlarkDict(map[string]any{"key": "value"})},
				{starlark.String("destination"), starlark.String(tt.destination)},
			}

			result, err := action.Run(ctx, "", "template_test", starlark.Tuple{}, kwargs)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.True(t, result.Success)

			// Verify destination file exists
			exists, err := afero.Exists(fs, tt.destination)
			assert.NoError(t, err)
			assert.True(t, exists)
		})
	}
}
