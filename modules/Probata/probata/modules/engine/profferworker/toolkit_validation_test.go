// Byline: Codex Â· GPT-6.1 Â· 2026-10-04. Validator wiring/config fixtures only; no live services or source processing.
package profferworker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
	"gopkg.in/yaml.v3"
)

// TestToolkitValidationOptionalConfiguration proves URL opt-in and exact store/service injection without I/O.
// Inputs: injected constructor fixtures. Outputs: enabled/disabled assertions. Effects: counters only.
// Choose instead of a real service constructor that would require mounted secrets and parser preflight.
func TestToolkitValidationOptionalConfiguration(t *testing.T) {
	stores, services := 0, 0
	store := smsthreads.S3Store{}
	service := &libraryvalidation.Service{}
	storeFactory := func() (libraryvalidation.VersionStore, error) { stores++; return store, nil }
	serviceFactory := func(got libraryvalidation.VersionStore) (*libraryvalidation.Service, error) {
		services++
		if !reflect.DeepEqual(got, store) {
			t.Fatal("B2 adapter replaced")
		}
		return service, nil
	}
	group, err := configureToolkitValidationWithFactories(context.Background(), "  ", storeFactory, serviceFactory)
	if err != nil || group != nil || stores != 0 || services != 0 {
		t.Fatal("disabled validator invoked constructors")
	}
	group, err = configureToolkitValidationWithFactories(context.Background(), "http://100.91.190.107:9077/mcp", storeFactory, serviceFactory)
	if err != nil || group == nil || group.Service != service || stores != 1 || services != 1 {
		t.Fatalf("configured admission: %v", err)
	}
}

// TestToolkitValidationConfigurationFailures rejects malformed endpoints and failed constructors without silent fallback.
// Inputs: independent malformed/canceled fixtures. Outputs: failure and call-boundary assertions. Effects: memory only.
// Choose for startup admission guards without B2, Surreal, NIM or Python calls.
func TestToolkitValidationConfigurationFailures(t *testing.T) {
	for _, kind := range []string{"relative-url", "other-scheme", "userinfo", "fragment", "store-error", "nil-store", "service-error", "nil-service", "canceled", "canceled-after-store", "canceled-after-service"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			stores, services := 0, 0
			endpoint := "http://100.91.190.107:9077/mcp"
			switch kind {
			case "relative-url":
				endpoint = "/mcp"
			case "other-scheme":
				endpoint = "file:///mcp"
			case "userinfo":
				endpoint = "http://synthetic:DO_NOT_ECHO@localhost/mcp"
			case "fragment":
				endpoint += "#invalid"
			case "canceled":
				cancel()
			}
			group, err := configureToolkitValidationWithFactories(ctx, endpoint, func() (libraryvalidation.VersionStore, error) {
				stores++
				if kind == "store-error" {
					return nil, errors.New("DO_NOT_ECHO")
				}
				if kind == "nil-store" {
					return nil, nil
				}
				if kind == "canceled-after-store" {
					cancel()
				}
				return smsthreads.S3Store{}, nil
			}, func(libraryvalidation.VersionStore) (*libraryvalidation.Service, error) {
				services++
				if kind == "service-error" {
					return nil, errors.New("DO_NOT_ECHO")
				}
				if kind == "nil-service" {
					return nil, nil
				}
				if kind == "canceled-after-service" {
					cancel()
				}
				return &libraryvalidation.Service{}, nil
			})
			if err == nil || group != nil || strings.Contains(err.Error(), "DO_NOT_ECHO") {
				t.Fatal("configured failure silently admitted or exposed credential")
			}
			if strings.HasPrefix(kind, "canceled") && !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation lost")
			}
			if (kind == "relative-url" || kind == "other-scheme" || kind == "userinfo" || kind == "fragment" || kind == "canceled") && stores != 0 {
				t.Fatal("invalid endpoint reached B2 constructor")
			}
			if (kind == "store-error" || kind == "nil-store" || kind == "canceled-after-store") && services != 0 {
				t.Fatal("failed B2 construction reached service")
			}
		})
	}
}

// TestToolkitValidationProductionConfigFailsBeforeRuntime checks the actual B2 config loader's fail-visible path.
// Inputs: invalid file environment, no credentials. Outputs: safe rejection. Effects: bounded config validation only.
// Choose to prove the production wrapper uses EnvB2ConfigFile rather than conversation credentials or a default store.
func TestToolkitValidationProductionConfigFailsBeforeRuntime(t *testing.T) {
	t.Setenv(libraryvalidation.EnvCaseMCPURL, "")
	if group, err := configureToolkitValidation(context.Background()); err != nil || group != nil {
		t.Fatal("unset URL was not disabled")
	}
	t.Setenv(libraryvalidation.EnvCaseMCPURL, "http://100.91.190.107:9077/mcp")
	t.Setenv(libraryvalidation.EnvB2ConfigFile, "relative-invalid.json")
	if group, err := configureToolkitValidation(context.Background()); err == nil || group != nil {
		t.Fatal("bad configured B2 file did not fail")
	}
}

// TestToolkitValidationRegistryPreservesCatalog adds exactly the validator workflow/four Activities alongside the catalog.
// Inputs: baseline catalog and injected validation groups. Outputs: registration set/count assertions. Effects: memory only.
// Choose over a full worker run to preserve all existing workflows without a Temporal server or database.
func TestToolkitValidationRegistryPreservesCatalog(t *testing.T) {
	catalog := activities.NewToolkitCatalogRegistrationActivities("", nil, &workerCatalogFixture{})
	baseline := &registrationRecorder{}
	RegisterAll(baseline, Registrations{ToolkitCatalog: &catalog})
	enabled := &registrationRecorder{}
	RegisterAll(enabled, Registrations{ToolkitCatalog: &catalog, ToolkitValidation: &activities.ToolkitLibraryValidationActivities{Service: &libraryvalidation.Service{}}})
	if enabled.workflowCount != baseline.workflowCount+1 || len(enabled.names) != len(baseline.names)+4 {
		t.Fatal("unexpected validator registry delta")
	}
	extra := map[string]bool{libraryvalidation.PrepareActivity: true, libraryvalidation.SnapshotActivity: true, libraryvalidation.VerifyActivity: true, libraryvalidation.ReceiptActivity: true}
	counts := map[string]int{}
	var rest []string
	for _, name := range enabled.names {
		if extra[name] {
			counts[name]++
		} else {
			rest = append(rest, name)
		}
	}
	for name := range extra {
		if counts[name] != 1 {
			t.Fatalf("validator Activity %s registered %d times", name, counts[name])
		}
	}
	sort.Strings(rest)
	prior := append([]string(nil), baseline.names...)
	sort.Strings(prior)
	if !reflect.DeepEqual(rest, prior) {
		t.Fatal("existing/catalog Activity registry changed")
	}
	found := 0
	var workflows []string
	for _, name := range enabled.workflowNames {
		if name == libraryvalidation.WorkflowName {
			found++
		} else {
			workflows = append(workflows, name)
		}
	}
	if found != 1 || !reflect.DeepEqual(workflows, baseline.workflowNames) {
		t.Fatal("existing/catalog workflow registry changed")
	}
}

// TestToolkitValidationDeploymentContract parses owned Compose configuration and checks the existing worker's baked runtime recipe.
// Inputs: exact deployment source files. Outputs: mount/env/parser-source assertions. Effects: file reads only.
// Choose before the parent builds/deploys the actual image; this does not claim a Docker runtime smoke test.
func TestToolkitValidationDeploymentContract(t *testing.T) {
	raw, err := os.ReadFile("../../../deploy/proffer-worker.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var compose struct {
		Services map[string]struct {
			Environment map[string]string `yaml:"environment"`
			Volumes     []string          `yaml:"volumes"`
		} `yaml:"services"`
	}
	if err = yaml.Unmarshal(raw, &compose); err != nil {
		t.Fatal(err)
	}
	cfg, ok := compose.Services["proffer-worker"]
	if !ok {
		t.Fatal("existing worker missing")
	}
	expected := map[string]string{libraryvalidation.EnvCaseMCPURL: "http://100.91.190.107:9077/mcp", libraryvalidation.EnvB2Bucket: "salem-data", libraryvalidation.EnvB2Prefix: "consignatio/casevault/DerivedKnowledge/family-court/library-validation", libraryvalidation.EnvB2ConfigFile: "/run/secrets/casebible-b2.json", libraryvalidation.EnvCaseTokenFile: "/run/secrets/case-mcp-token", libraryvalidation.EnvSigningKeyFile: "/run/secrets/validation-signing-key", libraryvalidation.EnvCurrencyKeyFile: "/run/secrets/currency-signing-key", libraryvalidation.EnvDBUserFile: "/run/secrets/validator-user", libraryvalidation.EnvDBPasswordFile: "/run/secrets/validator-password", "CASEBIBLE_RECOVERY_DATABASE_URL_FILE": "/run/secrets/casebible-recovery-database-url", libraryvalidation.EnvDBAccess: libraryvalidation.ValidatorAccess, "TOOLKIT_CURRENCY_DB_ACCESS": libraryvalidation.CurrencyAccess}
	for key, value := range expected {
		if cfg.Environment[key] != value {
			t.Fatalf("deployment mismatch for %s", key)
		}
	}
	if cfg.Environment[libraryvalidation.EnvDBUserFile] == cfg.Environment["SURREAL_CASE_USER_FILE"] || cfg.Environment[libraryvalidation.EnvDBPasswordFile] == cfg.Environment["SURREAL_CASE_PASSWORD_FILE"] {
		t.Fatal("validator reused conversation credentials")
	}
	mounts := map[string]bool{}
	for _, m := range cfg.Volumes {
		mounts[m] = true
	}
	for _, name := range []string{"case-mcp-token", "validation-signing-key", "currency-signing-key", "validator-user", "validator-password", "casebible-recovery-database-url", "currency-review-user", "currency-review-password", "currency-approval-key"} {
		if !mounts["/data/probata/secrets/family-court/"+name+":/run/secrets/"+name+":ro"] {
			t.Fatalf("missing dedicated read-only secret mount %s", name)
		}
	}
	if !mounts["/data/probata/volumes/proffer/library-validation:/data/proffer/library-validation"] {
		t.Fatal("private scratch bind missing")
	}
	raw, err = os.ReadFile("../../../deploy/docker/proffer-worker/Dockerfile")
	if err != nil {
		t.Fatal(err)
	}
	dockerfile := string(raw)
	for _, required := range []string{"FROM golang:1.26-bookworm AS builder", "FROM debian:bookworm-slim", "venv --copies", "--only-binary=:all: --no-deps", "USER 10001:10001", "--check-runtime", "-m 0700 /data/proffer/library-validation", "ENTRYPOINT [\"/usr/local/bin/proffer-worker\"]"} {
		if !strings.Contains(dockerfile, required) {
			t.Fatalf("missing existing-worker runtime step %s", required)
		}
	}
	for _, key := range []string{libraryvalidation.EnvPython, libraryvalidation.EnvProjectRoot, libraryvalidation.EnvBridgeFile, libraryvalidation.EnvScratchRoot} {
		if !strings.Contains(dockerfile, key+"=") {
			t.Fatalf("runtime path not baked: %s", key)
		}
	}
	if strings.Count(dockerfile, "\nFROM ") != 2 || strings.Contains(dockerfile, "PROFFER_WORKER_IMAGE") {
		t.Fatal("independent worker image introduced")
	}
	// Byline: Codex, 2026-10-04. Protected CLI shares the runtime; active provider routing is B2-only.
	if !strings.Contains(dockerfile, "COPY --from=builder /out/toolkit-currency-review /usr/local/bin/toolkit-currency-review") {
		t.Fatal("protected currency CLI missing from existing runtime")
	}
	if strings.Contains(cfg.Environment["OBJECT_STORES_JSON"], "r2") || strings.Contains(cfg.Environment["SOURCE_ROOTS_JSON"], "r2://") {
		t.Fatal("retired R2 remains an active worker route")
	}
	if cfg.Environment["TOOLKIT_CURRENCY_REVIEW_APPROVAL_KEY_FILE"] == cfg.Environment[libraryvalidation.EnvSigningKeyFile] || cfg.Environment["TOOLKIT_CURRENCY_REVIEW_APPROVAL_KEY_FILE"] == cfg.Environment[libraryvalidation.EnvCurrencyKeyFile] {
		t.Fatal("operator approval key reused a worker signing key")
	}
	for _, source := range []string{"server/__init__.py", "server/tools/__init__.py", "server/tools/registry.py", "server/tools/extractors/__init__.py", "server/tools/extractors/extract_text.py", "server/tools/extractors/html_text/__init__.py", "server/tools/extractors/html_text/_common.py", "server/tools/extractors/html_text/_ranks.py", "server/tools/extractors/html_text/lxml_html.py", "modules/engine/extraction/libraryvalidation/extractor_bridge.py", "modules/engine/extraction/libraryvalidation/runtime/requirements.txt"} {
		if _, err = os.Stat(filepath.Join("../../..", source)); err != nil {
			t.Fatalf("baked parser input absent: %s", source)
		}
	}
}
