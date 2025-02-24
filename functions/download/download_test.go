package download

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/discentem/starcm/functions/base"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"go.starlark.net/starlark"
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		action         *downloadAction
		moduleName     string
		starlarkArgs   starlark.Tuple
		starlarkKwargs []starlark.Tuple
		expectedResult func(*base.Result) bool
		expectedError  func(err error) bool
	}{
		{
			name: "Test download with nil fsys",
			action: &downloadAction{
				httpClient: &http.Client{},
				fsys:       nil,
			},
			moduleName: fmt.Sprintf("download %q", "hello world"),
			expectedResult: func(result *base.Result) bool {
				return result == nil
			},
			expectedError: func(err error) bool {
				return err != nil
			},
		},
		{
			name: "Test downloading succeeds",
			action: &downloadAction{
				httpClient: func() *http.Client {
					client := &http.Client{}
					client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(bytes.NewBufferString("hello world")),
						}, nil
					})
					return client
				}(),
				fsys: afero.NewMemMapFs(),
			},
			moduleName:   fmt.Sprintf("download %q", "hello world"),
			starlarkArgs: nil,
			starlarkKwargs: []starlark.Tuple{
				{
					starlark.String("url"),
					starlark.String("http://example.com"),
				},
				{
					starlark.String("save_to"),
					starlark.String("file.txt"),
				},
			},
			expectedResult: func(result *base.Result) bool {
				return result != nil
			},
			expectedError: func(err error) bool {
				return err == nil
			},
		},
		{
			name: "Test download with missing url",
			action: &downloadAction{
				httpClient: http.DefaultClient,
				fsys:       afero.NewMemMapFs(),
			},
			moduleName:   fmt.Sprintf("download %q", "hello world"),
			starlarkArgs: nil,
			starlarkKwargs: []starlark.Tuple{
				{
					starlark.String("save_to"),
					starlark.String("file.txt"),
				},
			},
			expectedResult: func(result *base.Result) bool {
				return result == nil
			},
			expectedError: func(err error) bool {
				return err != nil
			},
		},
		{
			name: "Test download with missing save_to",
			action: &downloadAction{
				httpClient: http.DefaultClient,
				fsys:       afero.NewMemMapFs(),
			},
			moduleName:   fmt.Sprintf("download %q", "hello world"),
			starlarkArgs: nil,
			starlarkKwargs: []starlark.Tuple{
				{
					starlark.String("url"),
					starlark.String("http://example.com"),
				},
			},
			expectedResult: func(result *base.Result) bool {
				return result == nil
			},
			expectedError: func(err error) bool {
				return err != nil
			},
		},
		{
			name: "Test non-200 status code",
			action: &downloadAction{
				httpClient: func() *http.Client {
					client := &http.Client{}
					client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusNotFound,
						}, nil
					})
					return client
				}(),
				fsys: afero.NewMemMapFs(),
			},
			moduleName:   fmt.Sprintf("download %q", "hello world"),
			starlarkArgs: nil,
			starlarkKwargs: []starlark.Tuple{
				{
					starlark.String("url"),
					starlark.String("http://example.com"),
				},
				{
					starlark.String("save_to"),
					starlark.String("file.txt"),
				},
			},
			expectedResult: func(result *base.Result) bool {
				return result == nil
			},
			expectedError: func(err error) bool {
				return err != nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Running test %q", tt.name)
			resultInterface, err := tt.action.Run(context.TODO(), "", tt.moduleName, tt.starlarkArgs, tt.starlarkKwargs)
			if tt.expectedError == nil {
				t.Fatal("tt.expectedError must be provided")
			}
			if tt.expectedResult == nil {
				t.Fatal("tt.expectedResult must be provided")
			}
			t.Logf("result: %v", resultInterface)

			var result *base.Result
			if resultInterface != nil {
				result = resultInterface.(*base.Result)
			}

			assert.Equal(t, true, tt.expectedError(err), "unexpected error")
			assert.Equal(t, true, tt.expectedResult(result), "unexpected result")

			if result != nil {
				if result.Success == true {
					t.Logf("result.Output: %q", *result.Output)
					assert.Equal(t, "downloaded file to file.txt", *result.Output)
					f, err := tt.action.fsys.Open("file.txt")
					assert.NoError(t, err)
					b, err := io.ReadAll(f)
					assert.NoError(t, err)
					t.Logf("file.txt content: %q", string(b))
					assert.Equal(t, "hello world", string(b))
				}
			}
		})
	}
}
