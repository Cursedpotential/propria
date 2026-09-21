// Byline: Claude Code · Opus 5 · 2026-09-20

package profferworker

import (
	"fmt"
	"sync"

	"github.com/Cursedpotential/probata/engine/acquisition"
	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/objectstores"
)

// deriveObjectStores resolves one configured locator scheme to a ready
// S3-compatible store. Clients are built on first use and reused: a store is
// only needed when a derive-routed source actually arrives, so a worker whose
// credentials for an unused provider are missing still starts.
type deriveObjectStores struct {
	stores objectstores.Stores

	mu     sync.Mutex
	cached map[string]smsthreads.ObjectStore
}

func newDeriveObjectStores(stores objectstores.Stores) *deriveObjectStores {
	return &deriveObjectStores{stores: stores, cached: map[string]smsthreads.ObjectStore{}}
}

func (d *deriveObjectStores) StoreForScheme(scheme string) (smsthreads.ObjectStore, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if store, ok := d.cached[scheme]; ok {
		return store, nil
	}
	credentialFile, ok := d.stores[scheme]
	if !ok {
		return nil, fmt.Errorf("scheme %q is not in %s (configured: %v)", scheme, objectstores.StoresEnv, d.stores.Schemes())
	}
	cfg, err := acquisition.LoadObjectStorageConfigFile(credentialFile)
	if err != nil {
		return nil, err
	}
	client, err := acquisition.NewS3Client(cfg)
	if err != nil {
		return nil, err
	}
	store := smsthreads.S3Store{Client: client}
	d.cached[scheme] = store
	return store, nil
}
