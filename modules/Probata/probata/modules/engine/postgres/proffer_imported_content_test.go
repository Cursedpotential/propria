package postgres

import (
	"strings"
	"testing"
)

func TestUploadContentDigestReadsOnlyAnExactUploadSHA256(t *testing.T) {
	valid := "upload://" + strings.Repeat("ab", 32)
	if digest := uploadContentDigest(valid); len(digest) != 32 || digest[0] != 0xab {
		t.Fatalf("digest = %x", digest)
	}
	for _, other := range []string{
		"b2://bucket/key?versionId=1", "upload://" + strings.Repeat("ab", 31), "upload://" + strings.Repeat("zz", 32), "", "upload://",
	} {
		if digest := uploadContentDigest(other); digest != nil {
			t.Fatalf("%q produced a digest %x", other, digest)
		}
	}
}
