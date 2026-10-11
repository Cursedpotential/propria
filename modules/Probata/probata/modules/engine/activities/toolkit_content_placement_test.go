// Byline: Codex · GPT-6 · 2026-10-04.
package activities

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

// placementFake reuses the preservation storage fixture and injects bounded retained-version observations.
// Inputs: synthetic versions and race switch; outputs: fake storage. Effects: memory only; choose for focused B2 contract tests.
type placementFake struct {
	*toolkitPreservationMemoryStore
	observations int
	race         bool
}

// PlacementVersions lists synthetic exact-key versions and optionally models an external concurrent arrival.
// Inputs: coordinates; outputs: retained IDs. Effects: controlled memory mutation; choose to test race evidence without network writes.
func (s *placementFake) PlacementVersions(_ context.Context, bucket, key string) ([]string, error) {
	s.observations++
	if s.race && s.observations == 2 {
		s.versions[bucket+"/"+key]["external-version"] = []byte("external")
	}
	out := []string{}
	for id := range s.versions[bucket+"/"+key] {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

// placementFixture retains tiny mounted sources and manifests beneath the explicit toolkit quarantine.
// Inputs: test; outputs: activity, request, fake store and root. Effects: creates bounded files with no cleanup/delete; choose instead of testing.T.TempDir.
func placementFixture(t *testing.T) (*ToolkitContentPlacementActivities, ToolkitContentPlacementInput, *placementFake, string) {
	t.Helper()
	root := retainedToolkitFixtureRoot(t, "toolkit-content-case-*")
	var err error
	if err = os.Mkdir(filepath.Join(root, "source"), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "source", "guide.md"), []byte("original text\n"), 0600); err != nil {
		t.Fatal(err)
	}
	m := ToolkitContentPlacementManifest{Units: []ToolkitContentPlacementUnit{{ID: "guide", SourceRef: toolkitFileRef(filepath.Join(root, "source")), Destination: "reference-data/family-court-toolkit", Files: []ToolkitContentPlacementFile{{Path: "guide.md", SHA256: digestBytes([]byte("original text\n")), Bytes: 14}}}}}
	data, _ := json.Marshal(m)
	if err = os.WriteFile(filepath.Join(root, "manifest.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	in := ToolkitContentPlacementInput{ManifestRef: toolkitFileRef(filepath.Join(root, "manifest.json")), ManifestSHA256: digestBytes(data), ReceiptRef: toolkitFileRef(filepath.Join(root, "receipt.json")), MaxFiles: 4, MaxFileBytes: 1024, MaxTotalBytes: 4096, MaxArchiveBytes: 4096}
	s := &placementFake{toolkitPreservationMemoryStore: &toolkitPreservationMemoryStore{objects: map[string][]byte{}, versions: map[string]map[string][]byte{}}}
	a := NewToolkitContentPlacementActivities(root, func(scheme string) (smsthreads.ObjectStore, error) {
		if scheme != "b2" {
			t.Fatalf("unexpected provider %s", scheme)
		}
		return s, nil
	})
	a.Heartbeat = nil
	return a, in, s, root
}

// placementRewriteManifest repins one synthetic manifest after intentional test mutation.
// Inputs: fixture root/request and transformation; outputs: new pinned request. Effects: rewrites only retained fixture metadata; choose for validation tests.
func placementRewriteManifest(t *testing.T, root string, in ToolkitContentPlacementInput, change func(*ToolkitContentPlacementManifest)) ToolkitContentPlacementInput {
	t.Helper()
	p := filepath.Join(root, "manifest.json")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var m ToolkitContentPlacementManifest
	if err = json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	change(&m)
	data, _ = json.Marshal(m)
	if err = os.WriteFile(p, data, 0600); err != nil {
		t.Fatal(err)
	}
	in.ManifestSHA256 = digestBytes(data)
	return in
}

// TestToolkitContentPlacementReplay proves retained exact-version placement and zero-write identical replay.
// Inputs: local tiny source and fake S3; outputs: source/destination/receipt assertions. Effects: retained test artifacts only; choose for provenance and idempotency coverage.
func TestToolkitContentPlacementReplay(t *testing.T) {
	a, in, s, root := placementFixture(t)
	r, err := a.placeToolkitContent(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Complete || len(r.Objects) != 1 || r.Objects[0].VersionID != "version-1" || r.ReceiptSHA256 == "" || r.Objects[0].ObjectKey != toolkitLegalPrefix+"reference-data/family-court-toolkit/guide.md" {
		t.Fatalf("unexpected receipt: %+v", r)
	}
	receipt, err := os.ReadFile(filepath.Join(root, "receipt.json"))
	if err != nil {
		t.Fatal(err)
	}
	if digestBytes(receipt) != r.ReceiptSHA256 {
		t.Fatal("receipt pin mismatch")
	}
	original, _ := os.ReadFile(filepath.Join(root, "source", "guide.md"))
	if string(original) != "original text\n" {
		t.Fatal("source changed")
	}
	r2, err := a.placeToolkitContent(context.Background(), in)
	if err != nil || !r2.Complete {
		t.Fatalf("replay failed: %v", err)
	}
	if s.versionedWrites != 1 || s.plainWrites != 0 || s.conditionalWrites != 0 {
		t.Fatal("replay wrote or unsafe writer used")
	}
}

// TestToolkitContentPlacementFailures rejects pin, body, path, provider and aggregate budget errors.
// Inputs: independently retained synthetic fixtures; outputs: visible failures without successful placement. Effects: tiny fixture writes; choose over implementation-mirroring unit checks.
func TestToolkitContentPlacementFailures(t *testing.T) {
	cases := []string{"pin", "source-hash", "file-budget", "total-budget", "traversal", "r2", "destination", "duplicate", "receipt-in-source", "root-escape"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			a, in, s, root := placementFixture(t)
			switch name {
			case "pin":
				in.ManifestSHA256 = digestBytes([]byte("wrong"))
			case "source-hash":
				if err := os.WriteFile(filepath.Join(root, "source", "guide.md"), []byte("mutated text!\n"), 0600); err != nil {
					t.Fatal(err)
				}
			case "file-budget":
				in.MaxFileBytes = 13
			case "total-budget":
				in.MaxTotalBytes = 13
			case "receipt-in-source":
				in.ReceiptRef = toolkitFileRef(filepath.Join(root, "source", "receipt.json"))
			default:
				in = placementRewriteManifest(t, root, in, func(m *ToolkitContentPlacementManifest) {
					switch name {
					case "traversal":
						m.Units[0].Files[0].Path = "../guide.md"
					case "r2":
						m.Units[0].SourceRef = "r2://retired/key?versionId=v"
					case "destination":
						m.Units[0].Destination = "recovery/library-sources"
					case "duplicate":
						m.Units[0].Files = append(m.Units[0].Files, m.Units[0].Files[0])
					case "root-escape":
						m.Units[0].SourceRef = toolkitFileRef(filepath.Dir(root))
					}
				})
			}
			if _, err := a.placeToolkitContent(context.Background(), in); err == nil {
				t.Fatal("expected failure")
			}
			if s.versionedWrites != 0 {
				t.Fatal("invalid input wrote destination")
			}
		})
	}
}

// TestToolkitContentPlacementChangedReplay rejects altered source, target version and receipt request identities.
// Inputs: prior successful placements and mutations; outputs: no replay writes. Effects: retained fixtures/memory only; choose for update detection evidence.
func TestToolkitContentPlacementChangedReplay(t *testing.T) {
	for _, which := range []string{"source", "remote", "request"} {
		t.Run(which, func(t *testing.T) {
			a, in, s, root := placementFixture(t)
			r, err := a.placeToolkitContent(context.Background(), in)
			if err != nil {
				t.Fatal(err)
			}
			switch which {
			case "source":
				err = os.WriteFile(filepath.Join(root, "source", "guide.md"), []byte("modified text\n"), 0600)
			case "remote":
				s.versions["salem-data/"+r.Objects[0].ObjectKey]["version-1"] = []byte("modified text\n")
			case "request":
				in.MaxFiles++
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err = a.placeToolkitContent(context.Background(), in); err == nil {
				t.Fatal("changed replay accepted")
			}
			if s.versionedWrites != 1 {
				t.Fatal("replay created another version")
			}
		})
	}
}

// TestToolkitContentPlacementConflict retains mismatch and concurrent-version evidence without rollback.
// Inputs: existing mismatch or intervening retained version; outputs: failed receipt containing observations. Effects: fake versions only; choose to prove explicitly non-atomic placement semantics.
func TestToolkitContentPlacementConflict(t *testing.T) {
	for _, race := range []bool{false, true} {
		t.Run(map[bool]string{false: "existing", true: "concurrent"}[race], func(t *testing.T) {
			a, in, s, root := placementFixture(t)
			key := "salem-data/" + toolkitLegalPrefix + "reference-data/family-court-toolkit/guide.md"
			if race {
				s.race = true
			} else {
				s.versions[key] = map[string][]byte{"version-1": []byte("different data")}
			}
			r, err := a.placeToolkitContent(context.Background(), in)
			if err == nil || r.Complete {
				t.Fatal("conflict accepted")
			}
			if len(r.Objects) != 1 || r.Objects[0].VersionID == "" {
				t.Fatal("missing conflict version identity")
			}
			if race {
				if len(r.Objects[0].AfterVersions) != 2 || len(s.versions[key]) != 2 {
					t.Fatal("concurrent versions not retained in receipt/store")
				}
			} else if s.versionedWrites != 0 {
				t.Fatal("existing mismatch wrote")
			}
			saved, err := os.ReadFile(filepath.Join(root, "receipt.json"))
			if err != nil {
				t.Fatal(err)
			}
			var retained ToolkitContentPlacementResult
			if err = json.Unmarshal(saved, &retained); err != nil {
				t.Fatal(err)
			}
			if retained.Error == "" {
				t.Fatal("failure hidden")
			}
		})
	}
}

// TestToolkitContentPlacementZIP streams one pinned archive once and checks member CRC and archive SHA.
// Inputs: tiny stored ZIP with two selected members, optionally corrupted; outputs: exact original members or failure. Effects: retained fake/fixture bytes only; choose to prove native bounded extraction without shell unzip.
func TestToolkitContentPlacementZIP(t *testing.T) {
	for _, mode := range []string{"valid", "archive-pin", "crc"} {
		t.Run(mode, func(t *testing.T) {
			a, in, s, root := placementFixture(t)
			var buf bytes.Buffer
			w := zip.NewWriter(&buf)
			for _, p := range []string{"nested/a.md", "nested/b.md"} {
				h := &zip.FileHeader{Name: p, Method: zip.Store}
				f, e := w.CreateHeader(h)
				if e != nil {
					t.Fatal(e)
				}
				if _, e = f.Write([]byte("original")); e != nil {
					t.Fatal(e)
				}
			}
			if e := w.Close(); e != nil {
				t.Fatal(e)
			}
			archive := append([]byte(nil), buf.Bytes()...)
			if mode == "crc" {
				at := bytes.Index(archive, []byte("original"))
				archive[at] = 'X'
			}
			s.versions["salem-data/recovery/source.zip"] = map[string][]byte{"source-version": archive}
			in = placementRewriteManifest(t, root, in, func(m *ToolkitContentPlacementManifest) {
				u := &m.Units[0]
				u.SourceRef = proffer.Ref("b2://salem-data/recovery/source.zip?versionId=source-version")
				u.ArchiveSHA256 = digestBytes(archive)
				if mode == "archive-pin" {
					u.ArchiveSHA256 = digestBytes([]byte("wrong"))
				}
				u.ArchiveBytes = int64(len(archive))
				u.Files = []ToolkitContentPlacementFile{{Path: "nested/a.md", SHA256: digestBytes([]byte("original")), Bytes: 8}, {Path: "nested/b.md", SHA256: digestBytes([]byte("original")), Bytes: 8}}
			})
			r, err := a.placeToolkitContent(context.Background(), in)
			if mode == "valid" {
				if err != nil || !r.Complete || len(r.Objects) != 2 {
					t.Fatalf("ZIP placement failed: %v", err)
				}
				reads := 0
				for _, v := range s.versionedReads {
					if v == "salem-data/recovery/source.zip#source-version" {
						reads++
					}
				}
				if reads != 1 {
					t.Fatal("archive read not deduplicated")
				}
			} else {
				if err == nil || s.versionedWrites != 0 {
					t.Fatal("ZIP corruption accepted")
				}
			}
		})
	}
}

// TestToolkitContentPlacementCancellation surfaces cancellation and preserves an attempted receipt without writes.
// Inputs: cancellation during bounded source reading; outputs: canceled error and retained partial receipt. Effects: fixture only; choose to verify heartbeat/cancellation path.
func TestToolkitContentPlacementCancellation(t *testing.T) {
	a, in, s, root := placementFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	a.Heartbeat = func(context.Context, ToolkitPackagePreservationHeartbeat) {
		calls++
		if calls == 2 {
			cancel()
		}
	}
	_, err := a.placeToolkitContent(ctx, in)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation: %v", err)
	}
	if s.versionedWrites != 0 {
		t.Fatal("canceled upload wrote")
	}
	if _, err = os.Stat(filepath.Join(root, "receipt.json")); err != nil {
		t.Fatal("partial receipt lost")
	}
}

// TestToolkitContentPlacementWorkflow proves stable registration identity and bounded summary flow.
// Inputs: mocked Activity and valid request; outputs: successful Temporal result. Effects: in-memory workflow only; choose for invocation contract validation.
func TestToolkitContentPlacementWorkflow(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	_, in, _, _ := placementFixture(t)
	env.RegisterActivityWithOptions(func(context.Context, ToolkitContentPlacementInput) (ToolkitContentPlacementSummary, error) {
		return ToolkitContentPlacementSummary{ReceiptRef: in.ReceiptRef, Complete: true, Objects: 1}, nil
	}, activity.RegisterOptions{Name: ToolkitContentPlacementActivityName})
	env.ExecuteWorkflow(ToolkitContentPlacementWorkflow, in)
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	var r ToolkitContentPlacementSummary
	if err := env.GetWorkflowResult(&r); err != nil || !r.Complete {
		t.Fatalf("workflow summary failed: %v", err)
	}
}

// TestToolkitContentPlacementVersionBudgets refuses incomplete version observations and disappeared retained versions.
// Inputs: synthetic observation sets; outputs: explicit conflict/budget failures. Effects: none; choose for bounded race-check contracts.
func TestToolkitContentPlacementVersionBudgets(t *testing.T) {
	if toolkitPlacementConflictChecks([]string{"old"}, []string{"new"}, "new", "new") == nil {
		t.Fatal("prior version loss accepted")
	}
	ids := make([]string, 65)
	if toolkitPlacementObservations(ids) == nil {
		t.Fatal("oversized observations accepted")
	}
}

// TestToolkitContentPlacementSpoolBudget bounds actual streamed bytes even when declared metadata is smaller.
// Inputs: oversized synthetic stream; outputs: rejection and retained partial spool. Effects: fixture scratch only; choose to verify runtime byte ceilings.
func TestToolkitContentPlacementSpoolBudget(t *testing.T) {
	_, _, _, root := placementFixture(t)
	f, err := toolkitPlacementBody(io.LimitReader(bytes.NewReader([]byte("oversized")), 9), ToolkitContentPlacementFile{Bytes: 2, SHA256: digestBytes([]byte("ov"))}, root, func() error { return nil })
	if f != nil {
		f.Close()
	}
	if err == nil {
		t.Fatal("stream budget ignored")
	}
}

// TestToolkitContentPlacementS3Race proves the existing SDK adapter pins its PUT and records another retained version.
// Inputs: local fake S3 HTTP endpoint with synthetic credentials and an intervening version; outputs: explicit race and both version IDs. Effects: loopback metadata/body requests only; choose for actual SDK request semantics beyond in-memory storage tests.
func TestToolkitContentPlacementS3Race(t *testing.T) {
	listings, puts := 0, 0
	payload := []byte("verified bytes")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.URL.Query()["versions"]; ok {
			if r.URL.Query().Get("max-keys") != "64" || r.URL.Query().Get("prefix") != "legal-key" {
				t.Error("unbounded or wrong version observation")
			}
			listings++
			w.Header().Set("Content-Type", "application/xml")
			if listings == 1 {
				fmt.Fprint(w, `<ListVersionsResult><IsTruncated>false</IsTruncated></ListVersionsResult>`)
			} else {
				fmt.Fprint(w, `<ListVersionsResult><IsTruncated>false</IsTruncated><Version><Key>legal-key</Key><VersionId>put-version</VersionId></Version><Version><Key>legal-key</Key><VersionId>external-version</VersionId></Version></ListVersionsResult>`)
			}
			return
		}
		switch r.Method {
		case http.MethodHead:
			if puts == 0 {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("x-amz-version-id", "put-version")
			w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
		case http.MethodPut:
			puts++
			data, err := io.ReadAll(r.Body)
			if err != nil || !bytes.Equal(data, payload) {
				t.Error("PUT payload changed")
			}
			w.Header().Set("x-amz-version-id", "put-version")
		case http.MethodGet:
			if r.URL.Query().Get("versionId") != "put-version" {
				t.Error("readback was not pinned")
			}
			w.Header().Set("x-amz-version-id", "put-version")
			w.Write(payload)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()
	cfg := aws.Config{Region: "test", Credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
		return aws.Credentials{AccessKeyID: "synthetic", SecretAccessKey: "synthetic"}, nil
	})}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) { o.BaseEndpoint = aws.String(server.URL); o.UsePathStyle = true })
	store := toolkitPlacementS3{smsthreads.S3Store{Client: client}}
	obj := ToolkitContentPlacementObject{ObjectKey: "legal-key", SHA256: digestBytes(payload), Bytes: int64(len(payload))}
	err := toolkitPlaceOne(context.Background(), store, bytes.NewReader(payload), &obj, false, nil, func() error { return nil })
	if err == nil || obj.VersionID != "put-version" || len(obj.AfterVersions) != 2 || obj.LatestVersionID != "put-version" || puts != 1 {
		t.Fatalf("SDK race not preserved: %v %+v puts=%d", err, obj, puts)
	}
}

// TestToolkitContentPlacementLocalZIP proves pinned mounted ZIPs reuse the bounded native archive/member path.
// Inputs: tiny retained local ZIP fixtures, pin/CRC/path/link/budget mutations and replay; outputs: authenticated placement or visible zero-write rejection. Effects: fixture files and fake B2 versions only; choose for server canonical ZIP admission without extraction or source alteration.
// Byline: Codex · GPT-6 · 2026-10-04.
func TestToolkitContentPlacementLocalZIP(t *testing.T) {
	for _, mode := range []string{"valid", "pin", "bytes", "crc", "member-path", "root-escape", "symlink", "archive-budget", "file-budget", "member-budget", "missing-pin", "replay"} {
		t.Run(mode, func(t *testing.T) {
			a, in, s, root := placementFixture(t)
			var buf bytes.Buffer
			writer := zip.NewWriter(&buf)
			for _, name := range []string{"nested/a.md", "nested/b.md"} {
				member, err := writer.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
				if err != nil {
					t.Fatal(err)
				}
				if _, err = member.Write([]byte("original")); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			archive := append([]byte(nil), buf.Bytes()...)
			if mode == "crc" {
				archive[bytes.Index(archive, []byte("original"))] = 'X'
			}
			source := filepath.Join(root, "canonical-toolkit-content-20261004.zip")
			if err := os.WriteFile(source, archive, 0600); err != nil {
				t.Fatal(err)
			}
			sourceRef := toolkitFileRef(source)
			if mode == "symlink" {
				link := filepath.Join(root, "linked.zip")
				if err := os.Symlink(source, link); err != nil {
					t.Skipf("host cannot create retained symlink fixture: %v", err)
				}
				sourceRef = toolkitFileRef(link)
			}
			if mode == "root-escape" {
				sourceRef = toolkitFileRef(filepath.Join(filepath.Dir(root), "outside.zip"))
			}
			in = placementRewriteManifest(t, root, in, func(m *ToolkitContentPlacementManifest) {
				u := &m.Units[0]
				u.SourceRef = sourceRef
				u.ArchiveSHA256 = digestBytes(archive)
				u.ArchiveBytes = int64(len(archive))
				u.Files = []ToolkitContentPlacementFile{{Path: "nested/a.md", SHA256: digestBytes([]byte("original")), Bytes: 8}, {Path: "nested/b.md", SHA256: digestBytes([]byte("original")), Bytes: 8}}
				switch mode {
				case "pin":
					u.ArchiveSHA256 = digestBytes([]byte("wrong"))
				case "bytes":
					u.ArchiveBytes++
				case "member-path":
					u.Files[0].Path = "../a.md"
				case "missing-pin":
					u.ArchiveSHA256 = ""
				}
			})
			switch mode {
			case "archive-budget":
				in.MaxArchiveBytes = int64(len(archive)) - 1
			case "file-budget":
				in.MaxFileBytes = 7
			case "member-budget":
				in.MaxFiles = 1
			}
			r, err := a.placeToolkitContent(context.Background(), in)
			if mode != "valid" && mode != "replay" {
				if err == nil || s.versionedWrites != 0 {
					t.Fatalf("invalid local ZIP accepted: %v", err)
				}
				if mode == "crc" && !strings.Contains(err.Error(), "checksum") {
					t.Fatalf("expected native CRC failure, got %v", err)
				}
				return
			}
			if err != nil || !r.Complete || len(r.Objects) != 2 {
				t.Fatalf("local ZIP failed: %v", err)
			}
			if len(s.versionedReads) != 2 {
				t.Fatal("local ZIP unexpectedly fetched from object storage")
			}
			untouched, err := os.ReadFile(source)
			if err != nil || !bytes.Equal(untouched, archive) {
				t.Fatal("original ZIP altered")
			}
			spools, err := filepath.Glob(filepath.Join(root, "placement-scratch-*", "placement-member-*"))
			if err != nil || len(spools) != 3 {
				t.Fatalf("expected one archive spool and two selected members, got %d: %v", len(spools), err)
			}
			replay, err := a.placeToolkitContent(context.Background(), in)
			if err != nil || !replay.Complete || s.versionedWrites != 2 {
				t.Fatalf("local ZIP replay wrote or failed: %v", err)
			}
			if mode == "replay" {
				changed := append([]byte(nil), archive...)
				changed[bytes.Index(changed, []byte("original"))] = 'Y'
				if err = os.WriteFile(source, changed, 0600); err != nil {
					t.Fatal(err)
				}
				if _, err = a.placeToolkitContent(context.Background(), in); err == nil || s.versionedWrites != 2 {
					t.Fatal("changed local ZIP replay accepted or wrote")
				}
			}
		})
	}
}
