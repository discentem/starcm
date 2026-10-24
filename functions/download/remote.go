package download

import (
	"context"
	"fmt"
	neturl "net/url"
	"path"
	"strings"
)

// Remote is a file named by one URL: http://, https:// or s3://bucket/key.
// load_remote() and install_remote_mac_pkg() take their source this way.
type Remote struct {
	// URL is the URL as given.
	URL string
	// Name is the base name of the URL's path, e.g. "agent.pkg".
	Name string
	// Folder is set when the URL's path ends in "/", the way a folder's
	// does. Fetching one needs Request.Folder too.
	Folder bool
	// S3 is the object for s3:// URLs, nil for http(s).
	S3 *S3Object
	// Signature locates the signature, "" when not pinned by one: a URL
	// for http(s), a key in the same bucket for S3.
	Signature string
}

// ParseRemote checks rawURL and, when it's pinned by one, its signature's
// URL. endpoint and region are only for s3:// URLs, and an S3 file's
// signature must be in the same bucket.
func ParseRemote(rawURL, signature, endpoint, region string) (Remote, error) {
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return Remote{}, err
	}
	r := Remote{URL: rawURL, Name: path.Base(u.Path), Folder: IsFolder(u.Path), Signature: signature}
	switch u.Scheme {
	case "http", "https":
		if endpoint != "" || region != "" {
			return Remote{}, fmt.Errorf("endpoint and region are only for s3:// URLs")
		}
	case "s3":
		key := strings.TrimPrefix(u.Path, "/")
		if u.Host == "" || key == "" {
			return Remote{}, fmt.Errorf("want s3://bucket/key, got %s", rawURL)
		}
		r.S3 = &S3Object{Bucket: u.Host, Key: key, Endpoint: endpoint, Region: region}
		if signature != "" {
			// FetchS3 takes the signature's key, in the same bucket.
			su, err := neturl.Parse(signature)
			if err != nil || su.Scheme != "s3" || su.Host != u.Host || strings.TrimPrefix(su.Path, "/") == "" {
				return Remote{}, fmt.Errorf("signature %s must be an s3:// URL in bucket %q", signature, u.Host)
			}
			r.Signature = strings.TrimPrefix(su.Path, "/")
		}
	default:
		return Remote{}, fmt.Errorf("url must start with http://, https:// or s3://, got %s", rawURL)
	}
	return r, nil
}

// FetchRemote downloads r as req describes, with FetchURL or FetchS3.
// r.Signature replaces req.Signature.
func (f *Fetcher) FetchRemote(ctx context.Context, r Remote, req Request) (Fetched, error) {
	req.Signature = r.Signature
	if r.S3 != nil {
		return f.FetchS3(ctx, *r.S3, req)
	}
	return f.FetchURL(ctx, r.URL, req)
}
