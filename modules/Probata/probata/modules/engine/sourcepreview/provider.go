package sourcepreview

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/Cursedpotential/probata/engine/acquisition"
	"github.com/Cursedpotential/probata/engine/objectstores"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

type ObjectClient interface {
	HeadObject(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}
type Provider struct {
	Clients map[string]ObjectClient
	Roots   objectstores.Roots
}

// NewProvider builds read-only clients using the existing object-store credential files and source roots.
// Inputs: existing store configuration and legacy credential paths. Outputs: range provider.
// Effects: reads configuration files only; choose for raw views instead of full immutable acquisition/sealing.
func NewProvider(stores objectstores.Stores, legacy map[string]string) (*Provider, error) {
	if len(stores) == 0 {
		stores = objectstores.Stores{}
		for k, v := range legacy {
			if v != "" {
				stores[k] = v
			}
		}
	}
	roots, err := objectstores.RootsFromEnv()
	if err != nil {
		return nil, ErrInvalid
	}
	p := &Provider{Clients: map[string]ObjectClient{}, Roots: roots}
	for _, scheme := range stores.Schemes() {
		cfg, err := acquisition.LoadObjectStorageConfigFile(stores[scheme])
		if err != nil {
			return nil, ErrUnavailable
		}
		client, err := acquisition.NewS3Client(cfg)
		if err != nil {
			return nil, ErrUnavailable
		}
		p.Clients[scheme] = client
	}
	return p, nil
}

// Open reads a pinned, bounded range from an authorized configured object store.
// Inputs: source locator, expected ETag/version and absolute offset. Outputs: exact source pins and stream.
// Effects: one HEAD and one conditional Range GET; choose only for pre-ingestion inspection, with no source writes.
func (p *Provider) Open(ctx context.Context, request Request, offset int64) (Source, io.ReadCloser, error) {
	u, err := url.Parse(request.SourceRef)
	if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || request.ExpectedETag == "" || len(request.ExpectedETag) > 512 || len(request.VersionID) > 512 {
		return Source{}, nil, ErrInvalid
	}
	key := strings.TrimPrefix(u.Path, "/")
	if key == "" || strings.Contains(key, "\\") {
		return Source{}, nil, ErrInvalid
	}
	for _, part := range strings.Split(key, "/") {
		if part == ".." {
			return Source{}, nil, ErrInvalid
		}
	}
	if _, ok := p.Roots.Match(u.Scheme, u.Host, key); !ok {
		return Source{}, nil, ErrInvalid
	}
	client := p.Clients[u.Scheme]
	if client == nil {
		return Source{}, nil, ErrUnavailable
	}
	headInput := &s3.HeadObjectInput{Bucket: aws.String(u.Host), Key: aws.String(key), IfMatch: aws.String(request.ExpectedETag)}
	if request.VersionID != "" {
		headInput.VersionId = aws.String(request.VersionID)
	}
	head, err := client.HeadObject(ctx, headInput)
	if err != nil {
		return Source{}, nil, providerError(err)
	}
	if head.ContentLength == nil || *head.ContentLength <= 0 || offset < 0 || offset >= *head.ContentLength || aws.ToString(head.ETag) != request.ExpectedETag || request.VersionID != "" && aws.ToString(head.VersionId) != request.VersionID {
		return Source{}, nil, ErrChanged
	}
	source := Source{SourceRef: request.SourceRef, ETag: aws.ToString(head.ETag), VersionID: aws.ToString(head.VersionId), ByteLength: *head.ContentLength}
	end := offset + MaxScanBytes - 1
	if end >= source.ByteLength {
		end = source.ByteLength - 1
	}
	input := &s3.GetObjectInput{Bucket: aws.String(u.Host), Key: aws.String(key), IfMatch: aws.String(source.ETag), Range: aws.String(fmt.Sprintf("bytes=%d-%d", offset, end))}
	if source.VersionID != "" {
		input.VersionId = aws.String(source.VersionID)
	}
	out, err := client.GetObject(ctx, input)
	if err != nil {
		return Source{}, nil, providerError(err)
	}
	if out.Body == nil {
		return Source{}, nil, ErrUnavailable
	}
	if aws.ToString(out.ETag) != source.ETag || source.VersionID != "" && aws.ToString(out.VersionId) != source.VersionID || out.ContentLength == nil || *out.ContentLength != end-offset+1 || aws.ToString(out.ContentRange) != fmt.Sprintf("bytes %d-%d/%d", offset, end, source.ByteLength) {
		out.Body.Close()
		return Source{}, nil, ErrChanged
	}
	return source, out.Body, nil
}

func providerError(err error) error {
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		switch apiError.ErrorCode() {
		case "PreconditionFailed", "NoSuchVersion", "NoSuchKey", "NotFound":
			return ErrChanged
		}
	}
	return ErrUnavailable
}
