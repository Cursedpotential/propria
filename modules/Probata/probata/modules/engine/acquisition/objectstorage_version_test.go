// Byline: Codex · GPT-6.1 · 2026-10-04.
package acquisition

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
)

type versionGetter struct {
	requested string
	returned  string
	calls     int
}

func (f *versionGetter) GetObject(_ context.Context, input *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	f.calls++
	f.requested = aws.ToString(input.VersionId)
	return &s3.GetObjectOutput{Body: io.NopCloser(bytes.NewReader([]byte("exact retained version"))), VersionId: aws.String(f.returned)}, nil
}

func TestVersionedObjectReferenceRequestsAndChecksExactProviderVersion(t *testing.T) {
	f := &versionGetter{returned: "retained+/version"}
	resolver, err := newObjectStorageAcquisitionResolver(t.TempDir(), "b2", f)
	require.NoError(t, err)
	_, err = resolver(t.Context(), proffer.Ref("b2://bucket/source.pdf?versionId=retained%2B%2Fversion"))
	require.NoError(t, err)
	require.Equal(t, "retained+/version", f.requested)
	f.returned = "wrong"
	_, err = resolver(t.Context(), proffer.Ref("b2://bucket/source.pdf?versionId=retained%2B%2Fversion"))
	require.ErrorContains(t, err, "different object version")
}

func TestVersionedObjectReferenceRejectsAmbiguousPinsBeforeGet(t *testing.T) {
	f := &versionGetter{}
	resolver, err := newObjectStorageAcquisitionResolver(t.TempDir(), "b2", f)
	require.NoError(t, err)
	for _, query := range []string{"versionId=", "versionId=null", "versionId=a&versionId=b", "versionId=%0A", "versionId=%ZZ"} {
		_, err = resolver(t.Context(), proffer.Ref("b2://bucket/source.pdf?"+query))
		require.Error(t, err, query)
	}
	require.Zero(t, f.calls)
	_, err = resolver(t.Context(), proffer.Ref("b2://bucket/source.pdf"))
	require.NoError(t, err)
	require.Empty(t, f.requested, "legacy references stay unversioned")
}
