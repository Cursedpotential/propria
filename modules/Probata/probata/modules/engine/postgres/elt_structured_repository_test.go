// Byline: Claude Code · Sonnet 5 · 2026-09-02
package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Cursedpotential/probata/engine/activities"
	"github.com/Cursedpotential/probata/engine/proffer"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestNewStructuredELTRepositoryRequiresDB(t *testing.T) {
	if _, err := NewStructuredELTRepository(nil); err == nil {
		t.Fatal("expected error when db is nil")
	}
}

func TestOpenStructuredELTRowsRejectsInvalidReferencesBeforeDatabaseAccess(t *testing.T) {
	acquireCalls := 0
	repo := newStructuredELTRepository(func(context.Context) (structuredELTSession, error) {
		acquireCalls++
		return nil, errors.New("database should not be reached")
	})
	valid := proffer.StageRequest{
		RequestID:        "workflow-1",
		SourceVersionRef: "11111111-1111-1111-1111-111111111111",
		DeclaredFormat:   "smsbackuprestore_xml",
		Refs:             map[string]proffer.Ref{"original": "22222222-2222-2222-2222-222222222222"},
	}
	for name, req := range map[string]proffer.StageRequest{
		"missing request": func() proffer.StageRequest { copy := valid; copy.RequestID = ""; return copy }(),
		"bad source":      func() proffer.StageRequest { copy := valid; copy.SourceVersionRef = "bad"; return copy }(),
		"missing original": func() proffer.StageRequest {
			copy := valid
			copy.Refs = map[string]proffer.Ref{}
			return copy
		}(),
		"bad original": func() proffer.StageRequest {
			copy := valid
			copy.Refs = map[string]proffer.Ref{"original": "bad"}
			return copy
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := repo.OpenStructuredELTRows(context.Background(), req, activities.StructuredELTFormatSMSXML); err == nil {
				t.Fatal("expected invalid reference to fail before database access")
			}
		})
	}
	if acquireCalls != 0 {
		t.Fatalf("invalid references acquired %d PostgreSQL sessions", acquireCalls)
	}
}

type webbedStatusRow struct {
	loaded bool
	err    error
}

func (r webbedStatusRow) Scan(destinations ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(destinations) != 1 {
		return errors.New("unexpected Webbed status destination count")
	}
	loaded, ok := destinations[0].(*bool)
	if !ok {
		return errors.New("Webbed status destination is not *bool")
	}
	*loaded = r.loaded
	return nil
}

type fakeStructuredELTSession struct {
	execSQL     string
	queryRowSQL string
	execErr     error
	status      webbedStatusRow
}

func (s *fakeStructuredELTSession) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	s.execSQL = sql
	return pgconn.CommandTag{}, s.execErr
}

func (*fakeStructuredELTSession) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected query")
}

func (s *fakeStructuredELTSession) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	s.queryRowSQL = sql
	return s.status
}

func (*fakeStructuredELTSession) Release() {}

func TestEnsureWebbedLoadedVerifiesWithoutFilesystemCalls(t *testing.T) {
	session := &fakeStructuredELTSession{status: webbedStatusRow{loaded: true}}
	if err := ensureWebbedLoaded(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	// A non-superuser session has LocalFileSystem disabled: neither an explicit
	// load nor duckdb_extensions() may be issued.
	if session.execSQL != "" {
		t.Fatalf("no statement may be executed to load Webbed: %s", session.execSQL)
	}
	if strings.Contains(session.queryRowSQL, "duckdb_extensions()") || strings.Contains(session.queryRowSQL, "load_extension") {
		t.Fatalf("readiness check touches the local filesystem: %s", session.queryRowSQL)
	}
	if !strings.Contains(session.queryRowSQL, "duckdb_functions()") || !strings.Contains(session.queryRowSQL, "function_name = 'read_xml'") {
		t.Fatalf("Webbed readiness was not verified from DuckDB: %s", session.queryRowSQL)
	}
}

func TestEnsureWebbedLoadedFailsClosed(t *testing.T) {
	for name, session := range map[string]*fakeStructuredELTSession{
		"verify failure": {status: webbedStatusRow{err: errors.New("status unavailable")}},
		"not loaded":     {status: webbedStatusRow{loaded: false}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ensureWebbedLoaded(context.Background(), session); err == nil {
				t.Fatal("expected Webbed readiness failure")
			}
		})
	}
}

func TestOnlyXMLTemplateRequiresWebbed(t *testing.T) {
	if !structuredELTRequiresWebbed(activities.StructuredELTFormatSMSXML) {
		t.Fatal("SMS XML must load Webbed")
	}
	for _, format := range []activities.StructuredELTFormat{
		activities.StructuredELTFormatCSV,
		activities.StructuredELTFormatNDJSON,
		activities.StructuredELTFormatChatGPTJSON,
		activities.StructuredELTFormatIMessageText,
	} {
		if structuredELTRequiresWebbed(format) {
			t.Fatalf("format %q unexpectedly requires Webbed", format)
		}
	}
}

func TestDuckDBSourceURLTranslatesR2ToS3(t *testing.T) {
	got, err := duckDBSourceURL("r2://nexus/proffer/test-fixtures/sample.xml")
	if err != nil {
		t.Fatal(err)
	}
	if got != "s3://nexus/proffer/test-fixtures/sample.xml" {
		t.Fatalf("duckDBSourceURL() = %q", got)
	}
}

func TestDuckDBSourceURLRejectsWorkerLocalOrUnretainedLocators(t *testing.T) {
	for _, locator := range []string{"upload://abc", "file:///sealed/source.xml", "b2://bucket/key"} {
		if _, err := duckDBSourceURL(locator); err == nil {
			t.Fatalf("expected %q to fail closed", locator)
		}
	}
}

func TestStructuredELTQueriesEmitCanonicalBundleColumns(t *testing.T) {
	formats := []activities.StructuredELTFormat{
		activities.StructuredELTFormatCSV,
		activities.StructuredELTFormatNDJSON,
		activities.StructuredELTFormatSMSXML,
		activities.StructuredELTFormatChatGPTJSON,
		activities.StructuredELTFormatIMessageText,
	}
	for _, format := range formats {
		query, err := structuredELTQuery(format, "s3://nexus/test/source")
		if err != nil {
			t.Fatalf("structuredELTQuery(%q) error = %v", format, err)
		}
		templateID, err := activities.StructuredELTTemplateForFormat(format)
		if err != nil {
			t.Fatalf("StructuredELTTemplateForFormat(%q) error = %v", format, err)
		}
		for _, column := range []string{"stored_bytes", "native_fields", "native_metadata"} {
			if !strings.Contains(query, column) {
				t.Fatalf("structuredELTQuery(%q) lacks %q output: %s", format, column, query)
			}
		}
		if !strings.Contains(query, "'duckdb_template', '"+templateID+"'") {
			t.Fatalf("structuredELTQuery(%q) does not emit pinned template %q", format, templateID)
		}
		if strings.Contains(strings.ToUpper(query), "INSERT ") {
			t.Fatalf("structuredELTQuery(%q) must not write raw tables", format)
		}
	}
}

func TestStructuredELTQueriesUseFormatSpecificDuckDBReaders(t *testing.T) {
	tests := map[activities.StructuredELTFormat][]string{
		activities.StructuredELTFormatSMSXML: {
			"read_xml(", "record_element := 'sms'", "record_element := 'mms'",
			"'$.address'", "'$.date'", "'$.type'", "'$.body'",
			"'$.parts.part[0].text'", "epoch_ms(", "'occurred_at'",
		},
		// payload-less MMS parts and failed decodes must survive normalization
		activities.StructuredELTFormatNDJSON:       {"'$.attachments'", "'$.attachment_references'", "'$.attachment_failures'"},
		activities.StructuredELTFormatChatGPTJSON:  {"read_text(", "json_each("},
		activities.StructuredELTFormatIMessageText: {"read_text(", "regexp_split_to_array("},
	}
	for format, expected := range tests {
		query, err := structuredELTQuery(format, "s3://nexus/test/source")
		if err != nil {
			t.Fatal(err)
		}
		for _, fragment := range expected {
			if !strings.Contains(query, fragment) {
				t.Fatalf("structuredELTQuery(%q) lacks %q", format, fragment)
			}
		}
	}
}

func TestStructuredELTQueryEscapesSourceURL(t *testing.T) {
	query, err := structuredELTQuery(activities.StructuredELTFormatChatGPTJSON, "https://example.invalid/o'brien.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "o''brien.json") {
		t.Fatalf("source URL was not escaped: %s", query)
	}
}

// TestFacebookMessengerQueryLinksAttachmentsBesideTheThreadFile proves the
// Messenger template emits the canonical three columns, the pinned template,
// the SMS-shaped message fields, and attachment locators under the thread
// file's own folder and scheme. Byline: Claude Code · Opus 5.5 · 2026-10-02
func TestFacebookMessengerQueryLinksAttachmentsBesideTheThreadFile(t *testing.T) {
	query, err := structuredELTQueryFor(activities.StructuredELTFormatFacebookMessenger,
		"s3://bucket/export/o'brien_1/message_1.json", "b2://bucket/export/o'brien_1/message_1.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"stored_bytes", "native_fields", "native_metadata", "'duckdb_template', 'facebook_messenger_json_v1'",
		"read_text('s3://bucket/export/o''brien_1/message_1.json')", "'b2://bucket/export/o''brien_1/' || attachment_paths",
		"'$.sender_name'", "'$.timestamp_ms'", "'$.photos[*].uri'", "'record_kind', 'message'", "'recipients'", "'attachments'",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("facebook messenger query lacks %q:\n%s", fragment, query)
		}
	}
	if strings.Contains(strings.ToUpper(query), "INSERT ") {
		t.Fatal("facebook messenger query must not write tables")
	}
	if _, err := structuredELTQueryFor(activities.StructuredELTFormatFacebookMessenger, "s3://bucket/x.json", "no-folder"); err == nil {
		t.Fatal("a source locator without a folder was accepted")
	}
	// Every other format is unchanged by the locator.
	plain, _ := structuredELTQuery(activities.StructuredELTFormatCSV, "s3://bucket/a.csv")
	routed, _ := structuredELTQueryFor(activities.StructuredELTFormatCSV, "s3://bucket/a.csv", "b2://bucket/a.csv")
	if plain != routed {
		t.Fatal("the locator changed a non-Messenger template")
	}
}

// HTML templates. Byline: Claude Code · Sonnet · 2026-10-02
func TestFacebookMessengerHTMLQueryResolvesMediaAgainstTheExportRoot(t *testing.T) {
	locator := "b2://bucket/Evidence/FB Exports/o'brien/facebook-x/your_facebook_activity/messages/inbox/thread_123/message_1.html"
	query, err := structuredELTQueryFor(activities.StructuredELTFormatFacebookMessengerHTML,
		"s3://bucket/Evidence/FB Exports/o'brien/facebook-x/your_facebook_activity/messages/inbox/thread_123/message_1.html", locator)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"'duckdb_template', 'facebook_messenger_html_v1'", "parse_html(content)", "'record_kind', 'message'",
		"'uri', 'b2://bucket/Evidence/FB Exports/o''brien/facebook-x/' || media_paths", "'thread_dir', 'thread_123'",
		"read_text('s3://bucket/Evidence/FB Exports/o''brien/facebook-x/your_facebook_activity/messages/inbox/thread_123/message_1.html')",
		"America/Detroit", "_a6-g", "_a6-p", "_a6-o", "_a6-q",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("facebook messenger html query lacks %q:\n%s", fragment, query)
		}
	}
	if strings.Contains(query, "{{") || strings.Contains(strings.ToUpper(query), "INSERT ") {
		t.Fatal("facebook messenger html query has an unfilled marker or writes a table")
	}
	// Without a marker folder the thread file's own folder is the root.
	query, err = structuredELTQueryFor(activities.StructuredELTFormatFacebookMessengerHTML, "s3://b/x/message_1.html", "b2://b/x/message_1.html")
	if err != nil || !strings.Contains(query, "'b2://b/x/' || media_paths") {
		t.Fatalf("fallback export root: %v\n%s", err, query)
	}
	if _, err := structuredELTQueryFor(activities.StructuredELTFormatFacebookMessengerHTML, "s3://b/x.html", "no-folder"); err == nil {
		t.Fatal("a source locator without a folder was accepted")
	}
}

func TestGenericHTMLDocumentQueryIsDocumentShaped(t *testing.T) {
	query, err := structuredELTQueryFor(activities.StructuredELTFormatGenericHTML, "s3://bucket/o'brien/page.html", "b2://bucket/o'brien/page.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"read_html_blocks('s3://bucket/o''brien/page.html')", "'duckdb_template', 'generic_html_document_v1'",
		"'record_kind', 'object'", "'doc_text'", "stored_bytes", "native_fields", "native_metadata",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("generic html query lacks %q:\n%s", fragment, query)
		}
	}
	// The text must never travel under the key the generic normalizer reads as a message body.
	if strings.Contains(query, "'body'") {
		t.Fatal("document text must not use the message body key")
	}
	if strings.Contains(query, "{{") || strings.Contains(strings.ToUpper(query), "INSERT ") {
		t.Fatal("generic html query has an unfilled marker or writes a table")
	}
	if !structuredELTRequiresWebbed(activities.StructuredELTFormatGenericHTML) || !structuredELTRequiresWebbed(activities.StructuredELTFormatFacebookMessengerHTML) {
		t.Fatal("HTML templates need the webbed extension check")
	}
}
