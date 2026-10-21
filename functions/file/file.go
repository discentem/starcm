package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/discentem/starcm/functions/base"
	"github.com/discentem/starcm/libraries/diffutils"
	shelllib "github.com/discentem/starcm/libraries/shell"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

type fileAction struct {
	fsys afero.Fs
	// executor runs pkgutil and codesign for file(signer = ...).
	executor shelllib.Executor
	goos     string
	// checkStored verifies source if it names a file in the local store.
	// Nil means no store.
	checkStored func(path string) error
}

var _ base.Runnable = (*fileAction)(nil)

func (a *fileAction) Run(
	ctx context.Context,
	workingDirectory string,
	label string,
	thread *starlark.Thread,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (*base.Result, error) {
	// Get action (create or delete)
	action, err := starlarkhelpers.FindValueInKwargsWithDefault(kwargs, "action", "create")
	if err != nil {
		return nil, fmt.Errorf("failed to find action in kwargs: %w", err)
	}

	if *action == "delete" {
		return a.runDelete(ctx, workingDirectory, label, thread, args, kwargs)
	}
	if starlarkhelpers.FindOptionalValueInKwargs(kwargs, "source") != nil {
		return a.runCopy(label, kwargs)
	}
	for _, name := range []string{"sha256", "signer"} {
		if starlarkhelpers.FindOptionalValueInKwargs(kwargs, name) != nil {
			return nil, fmt.Errorf("file(label=%q): %s needs source", label, name)
		}
	}

	return a.runCreate(ctx, workingDirectory, label, thread, args, kwargs)
}

func (a *fileAction) runCreate(
	_ context.Context,
	_ string,
	label string,
	_ *starlark.Thread,
	_ starlark.Tuple,
	kwargs []starlark.Tuple,
) (*base.Result, error) {
	if a.fsys == nil {
		return nil, fmt.Errorf("fsys must be provided to file module")
	}

	// Get path
	path, err := starlarkhelpers.FindValueinKwargs(kwargs, "path")
	if err != nil {
		return nil, err
	}
	if path == nil {
		return nil, fmt.Errorf("path must be provided to file(label=%q), cannot be nil", label)
	}

	// Expand homedir and resolve path relative to workspace (current working directory)
	filePath := *path
	filePath, err = homedir.Expand(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to expand home directory in path %q: %w", *path, err)
	}
	if !filepath.IsAbs(filePath) {
		filePath, err = filepath.Abs(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path %q: %w", *path, err)
		}
	}

	// Get optional parameters
	content, err := starlarkhelpers.FindValueInKwargsWithDefault(kwargs, "content", "")
	if err != nil {
		return nil, fmt.Errorf("failed to find content in kwargs: %w", err)
	}

	mode, err := starlarkhelpers.FindIntInKwargs(kwargs, "mode", 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to find mode in kwargs: %w", err)
	}

	createDirs, err := starlarkhelpers.FindBoolInKwargs(kwargs, "create_dirs", false)
	if err != nil {
		return nil, fmt.Errorf("failed to find create_dirs in kwargs: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	dirExists := true
	_, err = a.fsys.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			dirExists = false
		} else {
			return nil, fmt.Errorf("failed to stat directory %q: %w", dir, err)
		}
	}

	if !dirExists {
		if !createDirs {
			return nil, fmt.Errorf("directory %q does not exist and create_dirs is false", dir)
		}
		// Create intermediate directories
		if err := a.fsys.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directories %q: %w", dir, err)
		}
	}

	// Check if file exists with correct content
	fileExists := false
	var existingContent []byte
	if info, err := a.fsys.Stat(filePath); err == nil {
		fileExists = true
		if info.IsDir() {
			return nil, fmt.Errorf("%q is a directory, not a file", filePath)
		}
		// Read existing content
		data, err := afero.ReadFile(a.fsys, filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %q: %w", filePath, err)
		}
		existingContent = data
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to stat file %q: %w", filePath, err)
	}

	// If file exists with same content, nothing to do
	if fileExists && string(existingContent) == *content {
		return &base.Result{
			Label:   label,
			Message: func() *string { s := fmt.Sprintf("file %q already exists with correct content", filePath); return &s }(),
			Success: true,
			Changed: false,
		}, nil
	}

	// Write file
	f, err := a.fsys.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(mode))
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q for writing: %w", filePath, err)
	}
	defer f.Close()

	n, err := f.WriteString(*content)
	if err != nil {
		return nil, fmt.Errorf("failed to write to file %q: %w", filePath, err)
	}

	if n != len(*content) {
		return nil, fmt.Errorf("incomplete write to %q: wrote %d bytes out of %d", filePath, n, len(*content))
	}

	if err := f.Sync(); err != nil {
		return nil, fmt.Errorf("failed to sync file %q: %w", filePath, err)
	}

	// Generate diff if file existed before
	diff := ""
	if fileExists {
		diff = diffutils.GitDiff(string(existingContent), *content)
	}

	return &base.Result{
		Label:   label,
		Message: func() *string { s := fmt.Sprintf("created file %q", filePath); return &s }(),
		Success: true,
		Changed: true,
		Diff:    &diff,
	}, nil
}

func (a *fileAction) runDelete(
	_ context.Context,
	_ string,
	label string,
	_ *starlark.Thread,
	_ starlark.Tuple,
	kwargs []starlark.Tuple,
) (*base.Result, error) {
	if a.fsys == nil {
		return nil, fmt.Errorf("fsys must be provided to file module")
	}

	// Get path
	path, err := starlarkhelpers.FindValueinKwargs(kwargs, "path")
	if err != nil {
		return nil, err
	}
	if path == nil {
		return nil, fmt.Errorf("path must be provided to file(label=%q), cannot be nil", label)
	}

	// Expand homedir and resolve path relative to workspace (current working directory)
	filePath := *path
	filePath, err = homedir.Expand(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to expand home directory in path %q: %w", *path, err)
	}
	if !filepath.IsAbs(filePath) {
		filePath, err = filepath.Abs(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path %q: %w", *path, err)
		}
	}

	// Check if file exists and delete it
	_, err = a.fsys.Stat(filePath)
	if err == nil {
		// File exists, delete it
		if err := a.fsys.Remove(filePath); err != nil {
			return nil, fmt.Errorf("failed to delete file %q: %w", filePath, err)
		}
		return &base.Result{
			Label:   label,
			Message: func() *string { s := fmt.Sprintf("deleted file %q", filePath); return &s }(),
			Success: true,
			Changed: true,
		}, nil
	}

	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to stat file %q: %w", filePath, err)
	}

	// File doesn't exist, nothing to do
	return &base.Result{
		Label:   label,
		Message: func() *string { s := fmt.Sprintf("file %q does not exist", filePath); return &s }(),
		Success: true,
		Changed: false,
	}, nil
}

// Option configures file().
type Option func(*fileAction)

// WithStoreCheck makes file(source = ...) call check on source first, so a
// file from the local store is re-hashed before it's copied.
func WithStoreCheck(check func(path string) error) Option {
	return func(a *fileAction) { a.checkStored = check }
}

// WithExecutor replaces the executor file(signer = ...) runs pkgutil and
// codesign with, for tests.
func WithExecutor(executor shelllib.Executor) Option {
	return func(a *fileAction) { a.executor = executor }
}

// WithGOOS replaces the operating system file() thinks it runs on, for
// tests.
func WithGOOS(goos string) Option {
	return func(a *fileAction) { a.goos = goos }
}

func New(ctx context.Context, fsys afero.Fs, opts ...Option) *base.Module {
	a := &fileAction{
		fsys:     fsys,
		executor: &shelllib.RealExecutor{},
		goos:     runtime.GOOS,
	}
	for _, opt := range opts {
		opt(a)
	}
	return base.NewModule(
		ctx,
		"file",
		[]base.ArgPair{
			base.Arg[string]("path"),
			base.OptionalArg[string]("content"),
			base.OptionalArg[string]("action"),
			base.OptionalArgAllowNone[int64]("mode"),
			base.OptionalArgAllowNone[bool]("create_dirs"),
			base.OptionalArgAllowNone[string]("source"),
			base.OptionalArgAllowNone[string]("sha256"),
			base.OptionalArgAllowNone[string]("signer"),
			base.OptionalArgAllowNone[bool]("require_notarized"),
		},
		a,
	)
}
