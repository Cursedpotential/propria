import type { CasePerson } from "@/lib/case-identity-client";

export interface SourcePerspectiveChoice { scope: string; personId: string }

/** Find the single confirmed case owner from registry people, never from login or filename.
 * Input: registry people. Output: unique confirmed user or null. Effects: none.
 * Use for processing owner identity; source perspective still requires an explicit choice.
 */
export function confirmedCaseOwner(people: CasePerson[]): CasePerson | null {
  const owners = people.filter((person) => person.role_in_case === "user" && person.verification_state === "confirmed");
  return owners.length === 1 ? owners[0] : null;
}

/** Bind a phone/export choice to the exact selected source versions and mode.
 * Inputs: case mode and source/version tuples, or local acquisition request ID.
 * Output: stable selection key. Effects: none; changing a file/version invalidates the choice.
 */
export function sourcePerspectiveScope(mode: string, sources: Array<[string, string | null | undefined, number]>) {
  return JSON.stringify([mode, [...sources].sort((a, b) => a[0].localeCompare(b[0]))]);
}

/** Identify known human-message exports that already require downstream device perspective.
 * Inputs: source filename and chosen parser. Output: whether explicit perspective is required.
 * Effects: none. Generic XML/JSON, AI exports and non-message files gain no new processing gate.
 */
export function requiresMessagePerspective(name: string, format: string): boolean {
  return ["smsbackuprestore_xml", "callsbackuprestore_xml", "facebook_messenger_json", "facebook_messenger_html", "imessage_html"].includes(format)
    || (format === "xml" && /^(sms|calls)[-_].*\.xml$/i.test(name));
}

/** Resolve only an explicitly selected, current source perspective against registry people.
 * Inputs: people, current source scope, optional user choice and existing message requirement.
 * Output: processing identity fields. Effects: throws before submission if required facts are absent.
 * Use for normal Process; raw preview and AI context do not call this helper.
 */
export function processingIdentity(people: CasePerson[], scope: string, choice: SourcePerspectiveChoice | null, required: boolean) {
  const owner = confirmedCaseOwner(people);
  const person = choice?.scope === scope
    ? people.find((entry) => entry.id === choice.personId && entry.verification_state === "confirmed")
    : undefined;
  if (required && !owner) throw new Error("The case registry must identify one confirmed owner before processing this message export.");
  if (required && !person) throw new Error("Choose whose phone/export these selected messages came from.");
  return {
    ...(owner ? { owner_person_id: owner.id } : {}),
    ...(person ? { perspective_person_id: person.id } : {}),
  };
}
