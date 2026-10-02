// Byline: Claude Code · Sonnet · 2026-10-02
//
// People named by contact exports. Owner 2026-10-02 15:34: when a name is available a number is not a
// placeholder. The contacts import runs first: every contact becomes an UNCONFIRMED person
// (registry.person verification_state 'proposed', role 'unknown', requires_human_review true) whose
// display name is the most recent export's name, with every name any export gave kept as a candidate
// alias, the phones and emails as aliases, each marked "from contacts: <source key>". Placeholders are
// made afterwards, only for numbers no contact names (AddPlaceholders skips any carried number).
// Rows for the numbers are linked in the same transaction (only NULL entity columns). Every registry
// change appends registry.identity_change rows. Nothing is deleted or confirmed here.
package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Cursedpotential/probata/engine/caseidentity"
)

// AddContactPeople creates one unconfirmed person per contact. A number or email that any identifier
// already carries is left with its owner (and not duplicated); a contact whose numbers and emails are
// all carried is skipped. DryRun does everything and rolls back.
func (s *CaseIdentityStore) AddContactPeople(ctx context.Context, spec caseidentity.ContactPeopleSpec, actor caseidentity.Actor) (caseidentity.Receipt, error) {
	if err := caseidentity.ValidateContactPeople(spec); err != nil {
		return caseidentity.Receipt{}, err
	}
	if err := caseidentity.ValidateActor(actor); err != nil {
		return caseidentity.Receipt{}, err
	}
	key := actor.StoredKey("contact-people")
	tx, rollback, err := s.begin(ctx, "placeholders") // the same lock as placeholders: one batch at a time
	if err != nil {
		return caseidentity.Receipt{}, err
	}
	var already bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM registry.identity_change WHERE idempotency_key LIKE $1)`, key+":%").Scan(&already); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityError(err)
	}
	if already && !spec.DryRun {
		rollback()
		return caseidentity.Receipt{Ref: key, Kind: "registry.contact_people", RecordedAt: s.clock(), Replayed: true}, nil
	}

	type prepared struct {
		person  caseidentity.ContactPerson
		numbers []string
		emails  []string
		names   []string
	}
	var batch []prepared
	var lookups []string
	for _, person := range spec.People {
		p := prepared{person: person}
		seen := map[string]bool{}
		for _, raw := range person.Numbers {
			if number, ok := caseidentity.NormalizePhone(raw); ok && !seen[number] {
				seen[number] = true
				p.numbers = append(p.numbers, number)
				lookups = append(lookups, number)
			}
		}
		for _, raw := range person.Emails {
			email := strings.ToLower(strings.TrimSpace(raw))
			if strings.Contains(email, "@") && !seen[email] {
				seen[email] = true
				p.emails = append(p.emails, email)
				lookups = append(lookups, email)
			}
		}
		nameSeen := map[string]bool{}
		for _, name := range append([]string{person.DisplayName}, person.CandidateNames...) {
			name = strings.TrimSpace(name)
			if name != "" && !nameSeen[strings.ToLower(name)] {
				nameSeen[strings.ToLower(name)] = true
				p.names = append(p.names, name)
			}
		}
		batch = append(batch, p)
	}
	carriedRows, err := tx.Query(ctx, `SELECT DISTINCT registry.norm_identifier(a.alias_text::text)
		FROM registry.entity_alias a WHERE a.status <> 'retired' AND registry.norm_identifier(a.alias_text::text) = ANY($1::text[])`, lookups)
	if err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityError(err)
	}
	carried := map[string]bool{}
	for carriedRows.Next() {
		var value string
		if err := carriedRows.Scan(&value); err != nil {
			carriedRows.Close()
			rollback()
			return caseidentity.Receipt{}, caseIdentityError(err)
		}
		carried[value] = true
	}
	carriedRows.Close()
	if err := carriedRows.Err(); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityError(err)
	}

	at := s.clock()
	basis := func(source string) string { return "from contacts: " + source }
	var linkNumbers, linkEntities []string
	created, skipped, aliases := 0, 0, 0
	ids := map[string]string{}
	claimed := map[string]bool{} // a number or email given to a person earlier in this same batch
	for index, p := range batch {
		var freshNumbers, freshEmails []string
		for _, number := range p.numbers {
			if !carried[number] && !claimed[number] {
				freshNumbers = append(freshNumbers, number)
			}
		}
		for _, email := range p.emails {
			if !carried[email] && !claimed[email] {
				freshEmails = append(freshEmails, email)
			}
		}
		if len(freshNumbers) == 0 && len(freshEmails) == 0 {
			skipped++
			continue
		}
		personID, err := uuid.NewV7()
		if err != nil {
			rollback()
			return caseidentity.Receipt{}, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO registry.entity
			    (id, entity_type, display_name, canonical_name, data_tier, provenance, requires_human_review, review_status, safe_for_legal_use)
			VALUES ($1, 'person', $2, $2, 'inferred', ARRAY[ROW('postgres', $3::text, 'workbench.case_identity.contacts')::ai.source_ref],
			        true, 'unreviewed', false)`, personID, p.person.DisplayName, personID.String()); err != nil {
			rollback()
			return caseidentity.Receipt{}, caseIdentityWriteError(err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO registry.person (id, role_in_case, connection_to, notes, verification_state)
			VALUES ($1, 'unknown', 'unknown', $2, 'proposed')`, personID, basis(p.person.Source)); err != nil {
			rollback()
			return caseidentity.Receipt{}, caseIdentityWriteError(err)
		}
		var personAfter []byte
		if err := tx.QueryRow(ctx, personStateSQL, personID.String()).Scan(&personAfter); err != nil {
			rollback()
			return caseidentity.Receipt{}, caseIdentityError(err)
		}
		if _, err := insertChange(ctx, tx, "registry.person", personID.String(), []byte(`{}`), personAfter, spec.ChangeReason, actor,
			key+":p:"+personID.String(), at); err != nil {
			rollback()
			return caseidentity.Receipt{}, caseIdentityWriteError(err)
		}
		type alias struct{ text, kind, status string }
		var list []alias
		for _, number := range freshNumbers {
			list = append(list, alias{number, "phone", "confirmed"})
		}
		for _, email := range freshEmails {
			list = append(list, alias{email, "email", "candidate"})
		}
		for _, name := range p.names {
			list = append(list, alias{name, "name", "candidate"})
		}
		for _, a := range list {
			aliasID, err := uuid.NewV7()
			if err != nil {
				rollback()
				return caseidentity.Receipt{}, err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO registry.entity_alias
				    (id, entity_id, alias_text, alias_kind, status, period, basis, recorded_by, created_at, provenance)
				VALUES ($1, $2::uuid, $3, $4, $5, NULL, $6, $7, $8, ARRAY[ROW('postgres', $9::text, 'workbench.case_identity.contacts')::ai.source_ref])`,
				aliasID, personID.String(), a.text, a.kind, a.status, basis(p.person.Source), actor.Username, at, aliasID.String()); err != nil {
				rollback()
				return caseidentity.Receipt{}, caseIdentityWriteError(err)
			}
			var aliasAfter []byte
			var owner string
			if err := tx.QueryRow(ctx, aliasStateSQL, aliasID.String()).Scan(&aliasAfter, &owner); err != nil {
				rollback()
				return caseidentity.Receipt{}, caseIdentityError(err)
			}
			if _, err := insertChange(ctx, tx, "registry.entity_alias", aliasID.String(), []byte(`{}`), aliasAfter, spec.ChangeReason, actor,
				key+":a:"+aliasID.String(), at); err != nil {
				rollback()
				return caseidentity.Receipt{}, caseIdentityWriteError(err)
			}
			aliases++
		}
		for _, number := range freshNumbers {
			claimed[number] = true
			linkNumbers = append(linkNumbers, number)
			linkEntities = append(linkEntities, personID.String())
		}
		for _, email := range freshEmails {
			claimed[email] = true
		}
		created++
		if len(ids) < 20 {
			ids[fmt.Sprintf("%d", index)] = personID.String()
		}
	}

	linked := map[string]int64{}
	if len(linkNumbers) > 0 {
		for _, step := range linkRowsSQL {
			tag, err := tx.Exec(ctx, step.sql, linkNumbers, linkEntities)
			if err != nil {
				rollback()
				return caseidentity.Receipt{}, caseIdentityWriteError(err)
			}
			linked[step.name] = tag.RowsAffected()
		}
	}
	detail := map[string]any{
		"created": created, "skipped_all_carried": skipped, "aliases": aliases, "linked": linked,
		"dry_run": spec.DryRun, "entity_ids": ids,
	}
	if spec.DryRun {
		rollback()
		return caseidentity.Receipt{Ref: key, Kind: "registry.contact_people", RecordedAt: at, Detail: detail}, nil
	}
	if err := tx.Commit(ctx); err != nil {
		rollback()
		return caseidentity.Receipt{}, caseIdentityWriteError(err)
	}
	return caseidentity.Receipt{Ref: key, Kind: "registry.contact_people", RecordedAt: at, Detail: detail}, nil
}
