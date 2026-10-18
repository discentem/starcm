package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/discentem/starcm/functions/base"
	starlarkhelpers "github.com/discentem/starcm/starlark-helpers"
	"github.com/spf13/afero"
	"go.starlark.net/starlark"
)

// s3Options are s3_download()'s endpoint and region arguments. Neither is a
// secret; credentials always come from the AWS SDK's default chain
// (environment, shared config and credentials files, SSO, instance role).
type s3Options struct {
	endpoint string
	region   string
}

// defaultS3Region is used when neither region nor the AWS configuration
// names one. S3-compatible stores generally accept any region; AWS refuses
// requests for buckets in other regions, so those need region.
const defaultS3Region = "us-east-1"

// getS3 starts downloading s3://bucket/key with credentials from the AWS
// SDK's default chain, returning the object's body and size (-1 when
// unknown).
func getS3(ctx context.Context, bucket, key string, opts s3Options) (io.ReadCloser, int64, error) {
	var loadOpts []func(*config.LoadOptions) error
	if opts.region != "" {
		loadOpts = append(loadOpts, config.WithRegion(opts.region))
	}
	cfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, 0, fmt.Errorf("loading AWS configuration: %w", err)
	}
	if cfg.Region == "" {
		cfg.Region = defaultS3Region
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		// s3_download() checks the sha256 itself, so the SDK's note that an
		// object has no S3 checksum to validate is just noise.
		o.DisableLogOutputChecksumValidationSkipped = true
		if opts.endpoint != "" {
			o.BaseEndpoint = aws.String(opts.endpoint)
			// S3-compatible stores (MinIO and the like) address buckets in
			// the path, not the host name.
			o.UsePathStyle = true
		}
	})
	out, err := client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to download s3://%s/%s: %w", bucket, key, err)
	}
	size := int64(-1)
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	return out.Body, size, nil
}

// s3DownloadAction is s3_download(): download() with an S3 object instead of
// a URL. Saving, the sha256 check and the local store are shared.
type s3DownloadAction struct {
	*downloadAction
}

var _ base.Runnable = (*s3DownloadAction)(nil)

func (a *s3DownloadAction) Run(ctx context.Context, workingDirectory string, moduleName string, thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (*base.Result, error) {
	if a.fsys == nil {
		return nil, fmt.Errorf("fsys must be provided to s3_download module")
	}
	required := map[string]*string{}
	for _, k := range []string{"bucket", "key", "sha256"} {
		v, err := starlarkhelpers.FindValueinKwargs(kwargs, k)
		if err != nil {
			return nil, err
		}
		if v == nil || *v == "" {
			return nil, fmt.Errorf("s3_download(label=%q): %s must not be empty", moduleName, k)
		}
		required[k] = v
	}
	bucket, key, expectedHash := *required["bucket"], *required["key"], *required["sha256"]

	var opts s3Options
	for k, dst := range map[string]*string{"endpoint": &opts.endpoint, "region": &opts.region} {
		v, err := optionalString(kwargs, k)
		if err != nil {
			return nil, err
		}
		if v != nil {
			*dst = *v
		}
	}
	savePath, err := optionalString(kwargs, "save_to")
	if err != nil {
		return nil, err
	}
	liveProgress, err := starlarkhelpers.FindBoolInKwargs(kwargs, "live_progress", false)
	if err != nil {
		return nil, err
	}

	name := fmt.Sprintf("s3://%s/%s", bucket, key)
	get := func(ctx context.Context) (io.ReadCloser, error) {
		body, size, err := getS3(ctx, bucket, key, opts)
		if err != nil {
			return nil, err
		}
		return a.withProgress(body, size, name, liveProgress), nil
	}
	return a.save(ctx, moduleName, kwargs, savePath, path.Base(key), expectedHash, get, liveProgress)
}

// NewS3 returns the s3_download() builtin. It takes the same options as New,
// which configure the local store.
//
//	s3_download(label = "fetch agent", bucket = "releases", key = "agent.pkg", sha256 = "...")
func NewS3(ctx context.Context, fsys afero.Fs, writer io.Writer, opts ...Option) *base.Module {
	a := &downloadAction{
		httpClient: http.DefaultClient,
		fsys:       fsys,
		output:     writer,
		fn:         "s3_download",
	}
	for _, opt := range opts {
		opt(a)
	}
	return base.NewModule(
		ctx,
		"s3_download",
		[]base.ArgPair{
			base.Arg[string]("bucket"),
			base.Arg[string]("key"),
			base.Arg[string]("sha256"),
			base.OptionalArg[string]("save_to"),
			base.OptionalArg[string]("file_name"),
			base.OptionalArgAllowNone[bool]("live_progress"),
			base.OptionalArg[string]("endpoint"),
			base.OptionalArg[string]("region"),
		},
		&s3DownloadAction{a},
	)
}
