// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// Command entity-dryrun shows what "Extract entities" would propose for one
// run WITHOUT writing anything and without a database connection. It reads
// the read-only JSON-lines output of postgres.DryRunQueries on stdin and runs
// the production code in memory: DuckDB participant rows -> participant
// rules -> (optionally) model extraction on the first N messages ->
// reconcile -> supporting body mentions.
//
//	entity-dryrun sql <generation-id> [message-limit]     print the read-only SQL
//	psql -At ... | entity-dryrun propose [flags]          propose from stdin
//
// Flags for propose: --model-messages N (0 = rules only), --key-env-file F
// and --key-name NAME (read the model key from an env-style file; the value
// is never printed), --json (machine-readable output).
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Cursedpotential/probata/engine/extraction/entities"
	"github.com/Cursedpotential/probata/engine/extraction/events"
	"github.com/Cursedpotential/probata/engine/extraction/model"
	"github.com/Cursedpotential/probata/engine/postgres"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: entity-dryrun sql <generation-id> [message-limit] | entity-dryrun propose [flags]")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "sql":
		err = printSQL(os.Args[2:])
	case "propose":
		err = propose(os.Args[2:])
	case "probe":
		err = probe(os.Args[2:])
	case "sql-verify":
		printVerification()
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "entity-dryrun:", err)
		os.Exit(1)
	}
}

func printSQL(args []string) error {
	if len(args) < 1 {
		return errors.New("sql needs a generation id")
	}
	limit := 0
	if len(args) > 1 {
		if _, err := fmt.Sscan(args[1], &limit); err != nil {
			return err
		}
	}
	query, err := postgres.DryRunQueries(args[0], limit)
	if err != nil {
		return err
	}
	fmt.Print(query)
	return nil
}

// printVerification prints PREPARE statements for every write whose columns
// use ai.* types, inside a READ ONLY transaction that is rolled back.
func printVerification() {
	statements := postgres.VerificationStatements()
	names := make([]string, 0, len(statements))
	for name := range statements {
		names = append(names, name)
	}
	sort.Strings(names)
	fmt.Println("SET default_transaction_read_only = on;")
	for _, name := range names {
		// One transaction per statement so one failure never hides another.
		fmt.Printf("BEGIN READ ONLY;\n\\echo verify %s\nPREPARE verify_%s AS %s;\n\\echo prepared %s\nROLLBACK;\n", name, name, statements[name], name)
	}
}

type messageLine struct {
	RecordID            string                      `json:"record_id"`
	SourceVersionID     string                      `json:"source_version_id"`
	Ordinal             int64                       `json:"ordinal"`
	OccurredAt          *time.Time                  `json:"occurred_at"`
	SourceAvailableFrom *string                     `json:"source_available_from"`
	Body                string                      `json:"body"`
	Participants        []entities.MessageAddressee `json:"participants"`
}

type countingCompleter struct {
	inner   model.Completer
	calls   int
	elapsed time.Duration
}

func (c *countingCompleter) Complete(ctx context.Context, messages []model.Message, options model.CallOptions) (model.Completion, error) {
	started := time.Now()
	completion, err := c.inner.Complete(ctx, messages, options)
	c.calls++
	took := time.Since(started)
	c.elapsed += took
	if err == nil {
		thinking := "default"
		if options.Thinking != nil {
			thinking = fmt.Sprint(*options.Thinking)
		}
		fmt.Fprintf(os.Stderr, "  call %d: thinking=%s %.0fs finish=%q content_chars=%d reasoning_chars=%d prompt_tokens=%d completion_tokens=%d\n",
			c.calls, thinking, took.Seconds(), completion.FinishReason, len(completion.Content), completion.ReasoningChars, completion.PromptTokens, completion.CompletionTokens)
	}
	return completion, err
}

func propose(args []string) error {
	flags := flag.NewFlagSet("propose", flag.ContinueOnError)
	modelMessages := flags.Int("model-messages", 0, "run the model over the first N messages (0 = rules only)")
	keyEnvFile := flags.String("key-env-file", "", "env-style file holding the model key")
	keyName := flags.String("key-name", "NVIDIA_API_KEY", "variable name of the key in --key-env-file")
	asJSON := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}
	var rows []entities.ParticipantAggregate
	var messages []entities.MessageView
	sourceVersion := ""
	section := ""
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1<<20), 64<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "" || line == "SET" || line == "BEGIN" || line == "ROLLBACK":
			continue
		case strings.HasPrefix(line, "#"):
			section = line
			continue
		}
		switch section {
		case "#participants":
			var raw postgres.ParticipantRow
			if err := json.Unmarshal([]byte(line), &raw); err != nil {
				return fmt.Errorf("participant row: %w", err)
			}
			row, err := raw.Aggregate()
			if err != nil {
				return err
			}
			rows = append(rows, row)
		case "#messages":
			var raw messageLine
			if err := json.Unmarshal([]byte(line), &raw); err != nil {
				return fmt.Errorf("message row: %w", err)
			}
			sourceVersion = raw.SourceVersionID
			view := entities.MessageView{RecordID: raw.RecordID, Ordinal: raw.Ordinal, OccurredAt: raw.OccurredAt, Body: raw.Body, Participants: raw.Participants}
			if raw.SourceAvailableFrom != nil {
				if at, err := time.Parse(time.RFC3339Nano, *raw.SourceAvailableFrom); err == nil {
					view.SourceAvailableFrom = &at
				}
			}
			messages = append(messages, view)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	scope := entities.RunScope{GenerationID: "dry-run", SourceVersionID: sourceVersion}
	partials := entities.ProposeFromParticipants(scope, rows)
	for i := range partials {
		partials[i].CandidateID = fmt.Sprintf("rules-%d", i+1)
	}
	report := map[string]any{"participant_rows": len(rows), "messages": len(messages), "rules_proposals": len(partials)}
	var modelEvents []events.Proposal
	if *modelMessages > 0 {
		key, err := readKey(*keyEnvFile, *keyName)
		if err != nil {
			return err
		}
		cfg := model.Config{BaseURL: envOr(model.EnvBaseURL, model.DefaultBaseURL), ModelID: envOr(model.EnvModelID, model.DefaultModelID), APIKey: key, MaxTokens: model.DefaultMaxTokens}
		client, err := model.NewClient(cfg)
		if err != nil {
			return err
		}
		counter := &countingCompleter{inner: client}
		legend := legendFrom(rows, partials)
		slice := messages
		if len(slice) > *modelMessages {
			slice = slice[:*modelMessages]
		}
		var invalid []map[string]any
		firstTry, ungrounded := 0, 0
		batches := model.Batches(slice, legend, 0)
		for _, batch := range batches {
			outcome, err := model.ExtractBatch(context.Background(), counter, cfg.ModelID, batch, scope)
			if err != nil {
				return fmt.Errorf("batch %d: %w", batch.Index, err)
			}
			if outcome.Invalid {
				invalid = append(invalid, map[string]any{"batch": outcome.Index, "reason": outcome.Reason})
				continue
			}
			if outcome.Attempts == 1 {
				firstTry++
			}
			ungrounded += outcome.Ungrounded
			for i := range outcome.Entities {
				outcome.Entities[i].CandidateID = fmt.Sprintf("model-%d-%d", outcome.Index, i+1)
			}
			partials = append(partials, outcome.Entities...)
			modelEvents = append(modelEvents, outcome.Events...)
		}
		report["model"] = map[string]any{
			"model_id": cfg.ModelID, "batches": len(batches), "calls": counter.calls,
			"valid_first_try": firstTry, "invalid_after_retry": invalid, "ungrounded_items_dropped": ungrounded,
			"seconds_per_call": secondsPerCall(counter),
		}
	}
	plan := entities.Reconcile(entities.ReconcileInput{Scope: scope, Partials: partials})
	for i := range plan.Insert {
		body := 0
		for _, message := range messages {
			mentions := entities.BodyMentions(message, plan.Insert[i])
			body += len(mentions)
			plan.Insert[i].AddMentionSample(mentions...)
		}
		participant := 0
		for _, stat := range plan.Insert[i].Participants {
			participant += stat.MessageCount
		}
		plan.Insert[i].MentionCount = participant + body + len(plan.Insert[i].ModelMentions)
	}
	report["proposals"] = len(plan.Insert)
	report["events"] = len(modelEvents)
	if *asJSON {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"report": report, "entities": plan.Insert, "events": modelEvents})
	}
	printText(report, plan.Insert, modelEvents)
	return nil
}

// probe measures JSON conformance of the configured model on a fictional
// conversation (no case data leaves the machine).
func probe(args []string) error {
	flags := flag.NewFlagSet("probe", flag.ContinueOnError)
	calls := flags.Int("calls", 3, "number of extraction calls")
	keyEnvFile := flags.String("key-env-file", "", "env-style file holding the model key")
	keyName := flags.String("key-name", "NVIDIA_API_KEY", "variable name of the key")
	if err := flags.Parse(args); err != nil {
		return err
	}
	key, err := readKey(*keyEnvFile, *keyName)
	if err != nil {
		return err
	}
	cfg := model.Config{BaseURL: envOr(model.EnvBaseURL, model.DefaultBaseURL), ModelID: envOr(model.EnvModelID, model.DefaultModelID), APIKey: key, MaxTokens: model.DefaultMaxTokens}
	client, err := model.NewClient(cfg)
	if err != nil {
		return err
	}
	counter := &countingCompleter{inner: client}
	at := func(value string) *time.Time { parsed, _ := time.Parse(time.RFC3339, value); return &parsed }
	available := at("2026-09-21T01:14:39Z")
	lines := []struct{ from, to, when, body string }{
		{"self", "+15555550101", "2025-06-01T14:46:00Z", "Morning Kat, can you get Emma from Lincoln Elementary at 3?"},
		{"+15555550101", "self", "2025-06-01T14:50:00Z", "Yes. Tell Dr. Patel's office we moved her checkup to Thursday."},
		{"self", "+15555550101", "2025-06-01T15:02:00Z", "Catherine, the hearing at Oakland County court is on 2025-07-02 at 9:00."},
		{"+15555550101", "self", "2025-06-02T08:10:00Z", "Ok. Mom will drive Emma to the Walmart on Main St after school."},
		{"self", "+15555550101", "2025-06-02T08:15:00Z", "Thanks Katherine. I'll pay the $120 daycare bill Friday."},
	}
	var messages []entities.MessageView
	for i, line := range lines {
		messages = append(messages, entities.MessageView{
			RecordID: fmt.Sprintf("00000000-0000-4000-8000-%012d", i+1), Ordinal: int64(i), OccurredAt: at(line.when),
			SourceAvailableFrom: available, Body: line.body,
			Participants: []entities.MessageAddressee{{Role: "sender", Identifier: line.from}, {Role: "recipient", Identifier: line.to}},
		})
	}
	legend := []model.Participant{
		{Label: "P1", Address: entities.NormalizeAddress("self")},
		{Label: "P2", Address: entities.NormalizeAddress("+15555550101")},
	}
	scope := entities.RunScope{GenerationID: "probe", SourceVersionID: "probe"}
	firstTry, retried, invalid := 0, 0, 0
	var last model.BatchOutcome
	for i := 0; i < *calls; i++ {
		batch := model.Batches(messages, legend, i)[0]
		outcome, err := model.ExtractBatch(context.Background(), counter, cfg.ModelID, batch, scope)
		if err != nil {
			return err
		}
		switch {
		case outcome.Invalid:
			invalid++
			fmt.Printf("call %d: INVALID after retry: %s\n", i+1, outcome.Reason)
		case outcome.Attempts == 1:
			firstTry++
		default:
			retried++
		}
		for _, reason := range outcome.RetryReasons {
			fmt.Printf("call %d: retried because: %s\n", i+1, reason)
		}
		last = outcome
	}
	report, _ := json.MarshalIndent(map[string]any{
		"model_id": cfg.ModelID, "extractions": *calls, "http_calls": counter.calls,
		"valid_first_try": firstTry, "valid_after_one_retry": retried, "invalid_after_retry": invalid,
		"seconds_per_call": secondsPerCall(counter),
	}, "", "  ")
	fmt.Println("CONFORMANCE", string(report))
	fmt.Println("LAST OUTCOME (fictional data):")
	printText(map[string]any{"ungrounded": last.Ungrounded}, entities.Reconcile(entities.ReconcileInput{Scope: scope, Partials: withIDs(last.Entities)}).Insert, last.Events)
	return nil
}

func withIDs(list []entities.Proposal) []entities.Proposal {
	for i := range list {
		list[i].CandidateID = fmt.Sprintf("probe-%d", i+1)
	}
	return list
}

func secondsPerCall(counter *countingCompleter) float64 {
	if counter.calls == 0 {
		return 0
	}
	return float64(counter.elapsed.Milliseconds()) / 1000 / float64(counter.calls)
}

func legendFrom(rows []entities.ParticipantAggregate, proposals []entities.Proposal) []model.Participant {
	names := map[string]string{}
	for _, proposal := range proposals {
		if entities.LooksLikeAddress(proposal.Name) || strings.HasPrefix(proposal.Name, "Device owner") {
			continue
		}
		for _, alias := range proposal.AddressAliases() {
			names[string(alias.AddressKind)+":"+alias.Normalized] = proposal.Name
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if (rows[i].AddressKind == "self") != (rows[j].AddressKind == "self") {
			return rows[i].AddressKind == "self"
		}
		return rows[i].MessageCount > rows[j].MessageCount
	})
	var legend []model.Participant
	for i, row := range rows {
		address, _ := entities.AddressOf(row)
		legend = append(legend, model.Participant{Label: fmt.Sprintf("P%d", i+1), Address: address, Name: names[string(address.Kind)+":"+address.Normalized]})
	}
	return legend
}

var envLine = regexp.MustCompile(`^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.+?)\s*$`)

// readKey parses an env-style file with a tolerant regex (never `source`);
// the value is returned, never printed.
func readKey(path, name string) (string, error) {
	if path == "" {
		return "", errors.New("--key-env-file is required when --model-messages > 0")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		match := envLine.FindStringSubmatch(line)
		if match != nil && match[1] == name {
			return strings.Trim(match[2], `"'`), nil
		}
	}
	return "", fmt.Errorf("%s not found in %s", name, path)
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func printText(report map[string]any, proposals []entities.Proposal, found []events.Proposal) {
	encoded, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println("REPORT", string(encoded))
	fmt.Printf("\nPROPOSED ENTITIES (%d)\n", len(proposals))
	for _, proposal := range proposals {
		fmt.Printf("- %s [%s] mentions=%d confidence=%.2f extractors=%s\n", proposal.Name, proposal.RegistryType, proposal.MentionCount, proposal.Confidence, strings.Join(proposal.Extractors, ","))
		for _, alias := range proposal.Aliases {
			scope := ""
			if alias.Scope != "" {
				scope = " (scoped to " + alias.Scope + ", never a registry alias)"
			}
			fmt.Printf("    alias %-28q kind=%-11s source=%s%s\n", alias.Text, alias.Kind, alias.Source, scope)
		}
		for i, mention := range proposal.MentionSample {
			if i == 3 {
				break
			}
			fmt.Printf("    mention #%d %s %s %q %s\n", mention.Ordinal, mention.Role, mention.Kind, mention.Surface, mention.Snippet)
		}
		for _, flag := range proposal.Flags {
			fmt.Printf("    flag %s: %s\n", flag.Code, flag.Detail)
		}
	}
	fmt.Printf("\nPROPOSED EVENTS (%d)\n", len(found))
	for _, event := range found {
		when := ""
		if event.OccurredAt != nil {
			when = event.OccurredAt.UTC().Format("2006-01-02 15:04")
		}
		available := ""
		if at := event.SourceAvailableFrom(); at != nil {
			available = at.UTC().Format("2006-01-02")
		}
		source := ""
		if len(event.SourceRecords) > 0 {
			source = fmt.Sprintf("#%d", event.SourceRecords[0].Ordinal)
		}
		fmt.Printf("- %s | %s | %s (%s) stated=%q | people=%s | source %s | visible from %s\n",
			event.Title, event.EventType, when, event.TemporalPrecision, event.WhenStated, strings.Join(event.EntityNames, ","), source, available)
	}
}
