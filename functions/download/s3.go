package download

import (
	"context"
	"fmt"
	"io"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/discentem/starcm/functions/base"
)

// defaultS3Region is used when neither region nor the AWS configuration
// names one. S3-compatible stores generally accept any region; AWS refuses
// requests for buckets in other regions, so those need region.
const defaultS3Region = "us-east-1"

// getS3 starts downloading s3://bucket/key with credentials from the AWS
// SDK's default chain, returning the object's body and size (-1 when
// unknown).
func getS3(ctx context.Context, bucket, key string, obj S3Object) (io.ReadCloser, int64, error) {
	var loadOpts []func(*config.LoadOptions) error
	if obj.Region != "" {
		loadOpts = append(loadOpts, config.WithRegion(obj.Region))
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
		if obj.Endpoint != "" {
			o.BaseEndpoint = aws.String(obj.Endpoint)
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

// FetchS3 downloads obj as req describes, with credentials from the AWS
// SDK's default chain (environment, shared config and credentials files,
// SSO, instance role). req.Signature is the key of the signature file, in
// the same bucket. A key ending in "/" is a folder, fetched with
// FetchFolder.
func (f *Fetcher) FetchS3(ctx context.Context, obj S3Object, req Request) (Fetched, error) {
	if f.fsys == nil {
		return Fetched{}, fmt.Errorf("fsys must be provided to s3_download module")
	}
	if req.Function == "" {
		req.Function = "s3_download"
	}
	if obj.Bucket == "" {
		return Fetched{}, req.errorf("bucket must not be empty")
	}
	if obj.Key == "" {
		return Fetched{}, req.errorf("key must not be empty")
	}
	if IsFolder(obj.Key) {
		return f.FetchFolder(ctx, Source{S3: &obj}, req)
	}
	expectedHash, sig, err := f.pin(ctx, req, func(ctx context.Context, sigKey string) (io.ReadCloser, error) {
		body, _, err := getS3(ctx, obj.Bucket, sigKey, obj)
		return body, err
	})
	if err != nil {
		return Fetched{}, err
	}
	name := fmt.Sprintf("s3://%s/%s", obj.Bucket, obj.Key)
	get := func(ctx context.Context) (io.ReadCloser, error) {
		body, size, err := getS3(ctx, obj.Bucket, obj.Key, obj)
		if err != nil {
			return nil, err
		}
		return f.withProgress(body, size, name, req.LiveProgress), nil
	}
	return f.save(ctx, req, path.Base(obj.Key), expectedHash, sig, get)
}

// s3DownloadAction is s3_download(): download() with an S3 object instead of
// a URL.
type s3DownloadAction struct {
	f *Fetcher
}

var _ base.Action = (*s3DownloadAction)(nil)

func (a *s3DownloadAction) Execute(inst *base.FunctionInstance) (*base.Result, error) {
	var obj S3Object
	obj.Bucket, _ = base.Get[string](inst, "bucket")
	obj.Key, _ = base.Get[string](inst, "key")
	obj.Endpoint, _ = base.Get[string](inst, "endpoint")
	obj.Region, _ = base.Get[string](inst, "region")
	req, err := requestFrom(inst, "s3_download")
	if err != nil {
		return nil, err
	}
	got, err := a.f.FetchS3(inst.Ctx, obj, req)
	if err != nil {
		return nil, err
	}
	return resultFrom(inst.Label, got), nil
}

// NewS3 returns the s3_download() builtin.
//
//	s3_download(label = "fetch agent", bucket = "releases", key = "agent.pkg", sha256 = "...")
func NewS3(ctx context.Context, f *Fetcher) *base.Module {
	return base.NewModule(
		ctx,
		"s3_download",
		[]base.ArgPair{
			base.Arg[string]("bucket"),
			base.Arg[string]("key"),
			base.OptionalArg[string]("sha256"),
			base.OptionalArg[string]("signature"),
			base.OptionalArg[string]("save_to"),
			base.OptionalArg[string]("file_name"),
			base.OptionalArgAllowNone[bool]("live_progress"),
			base.OptionalArg[string]("endpoint"),
			base.OptionalArg[string]("region"),
		},
		&s3DownloadAction{f: f},
	)
}
