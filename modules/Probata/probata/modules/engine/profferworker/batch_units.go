package profferworker

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"sync"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/objectstores"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/Cursedpotential/probata/engine/sourcepreview"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// batchUnitProvider binds lazy authorized HEAD and pinned range reads to the existing source-preview provider.
// Inputs: existing credential-file map. Outputs: metadata-only Activity adapters with configured-root checks.
// Effects: reads credential files on first use and performs bounded provider reads; no new storage service.
func batchUnitProvider(stores objectstores.Stores) (activities.BatchUnitHead, activities.BatchUnitRead, func(string) error) {
	var mu sync.Mutex
	providers := map[string]*sourcepreview.Provider{}
	provider := func(ref string) (*sourcepreview.Provider, *url.URL, error) {
		u, err := url.Parse(ref)
		if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return nil, nil, errors.New("package source locator is invalid")
		}
		mu.Lock()
		defer mu.Unlock()
		p := providers[u.Scheme]
		if p == nil {
			file, ok := stores[u.Scheme]
			if !ok {
				return nil, nil, errors.New("package source store is not configured")
			}
			p, err = sourcepreview.NewProvider(objectstores.Stores{u.Scheme: file}, nil)
			if err != nil {
				return nil, nil, err
			}
			providers[u.Scheme] = p
		}
		key := strings.TrimPrefix(u.Path, "/")
		if _, ok := p.Roots.Match(u.Scheme, u.Host, key); !ok {
			return nil, nil, errors.New("package source is outside configured roots")
		}
		return p, u, nil
	}
	authorize := func(ref string) error {
		u, err := url.Parse(ref)
		if err != nil {
			return err
		}
		u.Path += "\x00" // The existing folder authorization convention admits a configured root itself.
		_, _, err = provider(u.String())
		return err
	}
	head := func(ctx context.Context, ref string) (proffer.BatchUnitMember, error) {
		p, u, err := provider(ref)
		if err != nil {
			return proffer.BatchUnitMember{}, err
		}
		out, err := p.Clients[u.Scheme].HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(u.Host), Key: aws.String(strings.TrimPrefix(u.Path, "/"))})
		if err != nil {
			return proffer.BatchUnitMember{}, err
		}
		if out.ContentLength == nil {
			return proffer.BatchUnitMember{}, errors.New("package object has no byte length")
		}
		return proffer.BatchUnitMember{SourceRef: ref, ProviderVersionID: aws.ToString(out.VersionId), ETag: aws.ToString(out.ETag), ByteLength: *out.ContentLength}, nil
	}
	read := func(ctx context.Context, member proffer.BatchUnitMember) ([]byte, error) {
		p, _, err := provider(member.SourceRef)
		if err != nil {
			return nil, err
		}
		source, stream, err := p.Open(ctx, sourcepreview.Request{SourceRef: member.SourceRef, ExpectedETag: member.ETag, VersionID: member.ProviderVersionID}, 0)
		if err != nil {
			return nil, err
		}
		defer stream.Close()
		if source.ByteLength != member.ByteLength {
			return nil, errors.New("package entrypoint length changed")
		}
		return io.ReadAll(io.LimitReader(stream, sourcepreview.MaxScanBytes))
	}
	return head, read, authorize
}
