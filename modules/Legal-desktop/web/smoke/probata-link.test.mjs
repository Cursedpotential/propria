import assert from "node:assert/strict";
import { test } from "node:test";
import { parseProbataLink } from "../src/lib/probata-link.ts";

const identity = "A5AAF531-9FB9-4D35-A56E-A41B2A8A2D60";
test("native links preserve kind and canonical identity", () => {
  for (const kind of ["entity", "event"]) {
    assert.deepEqual(parseProbataLink(`?probata_kind=${kind}&probata_id=${identity}`), {
      kind, recordId: identity.toLowerCase(),
    });
  }
});
test("incomplete, candidate and malformed identities do not resolve", () => {
  for (const search of ["", `?probata_id=${identity}`, "?probata_kind=entity",
    `?probata_kind=candidate&probata_id=${identity}`, "?probata_kind=entity&probata_id=proposal:123",
    "?probata_kind=event&probata_id=../../record", "?probata_kind=entity&probata_id=00000000-0000-0000-0000-000000000000"]) {
    assert.equal(parseProbataLink(search), null);
  }
});
