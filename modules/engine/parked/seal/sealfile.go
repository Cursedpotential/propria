//go:build parked

// PARKED 2026-09-06 (owner ruling 11:16/11:22): sealing is custody-at-PROMOTION work,
// not an ingest-time step. Keep for reuse when the promote activity is built;
// do not rewrite. Excluded from `go build ./...` by the build tag above.
//
// Byline: Claude Code · Fable 5.1 · 2026-09-06 (manual seal of a local file)
//
// SealLocalFile: read one file, seal it into the content-addressed store every
// other provider publishes into, return the upload://<sha256> reference the
// platform already resolves. No host-to-host data movement here.
package acquisition

import (
	"context"
	"encoding/hex"
	"fmt"

	platformpostgres "github.com/Cursedpotential/probata/engine/postgres"
	"github.com/Cursedpotential/probata/engine/proffer"
)

// SealLocalFile seals the regular file at path into root (the same seal root
// the upload ingress and the acquisition resolvers use) and returns the
// sealed acquisition plus its upload:// reference. The source file is never
// modified or moved. Sealing is content-addressed and idempotent: sealing the
// same bytes twice yields the same reference and leaves one object.
func SealLocalFile(ctx context.Context, root, path string) (platformpostgres.ImmutableAcquisition, proffer.Ref, error) {
	if _, err := prepareSealRoot(root); err != nil {
		return platformpostgres.ImmutableAcquisition{}, "", err
	}
	source, err := openRegularNonAliasFile(path)
	if err != nil {
		return platformpostgres.ImmutableAcquisition{}, "", fmt.Errorf("acquisition: open %s: %w", path, err)
	}
	defer source.Close()
	sealed, err := sealStream(ctx, root, source)
	if err != nil {
		return platformpostgres.ImmutableAcquisition{}, "", fmt.Errorf("acquisition: seal %s: %w", path, err)
	}
	ref := proffer.Ref(uploadRefScheme + "://" + hex.EncodeToString(sealed.ContentSHA256))
	return sealed, ref, nil
}
