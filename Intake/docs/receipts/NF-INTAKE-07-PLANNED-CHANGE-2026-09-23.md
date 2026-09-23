# NF-INTAKE-07 planned change

**Recorded:** 2026-09-23
**Repository / baseline:** Consignatio `main@99899c3aba7adf4717ffbcac809092608952e7a6`
**Status:** implemented and locally verified; deployment configuration held

## Finding reproduced

`src/features/live-selection/use-intake-selection-bridge.ts` registers a browser
`message` listener that validates message data but, at this baseline, does not
validate `event.origin` or `event.source`. `src/app/App.tsx` mounts the live
selection panel that activates this bridge. No matching bridge test was present.

## Bounded change

- Claim `src/features/live-selection/use-intake-selection-bridge.ts`, its
  focused test file, and this receipt only. `src/app/App.tsx` will be changed
  only if the existing wiring cannot supply the explicit trusted origin/source
  configuration without it.
- Add an explicit configuration-driven trusted-origin and expected-source
  contract. Accept a message only when configuration is present, the event
  origin exactly matches it, the event source is the configured expected source,
  and the existing payload validation succeeds.
- Fail closed if browser/Tauri configuration is absent or invalid. Do not infer
  authority from a host suffix, a tailnet name, a wildcard, or a message field.
- Preserve valid selection updates. Verify positive allowed origin/source plus
  rejected foreign origin, wrong source, malformed payload, and absent
  configuration, each without state side effects.

## Local verification

- `npm run test -- src/features/live-selection/use-intake-selection-bridge.test.ts --maxWorkers=1`
  — passed: 1 file, 5 tests.
- `npm run build` — passed: TypeScript project build and Vite production build.
- The focused tests prove the exact allow/reject cases above by inspecting the
  parsed event before state application. They do not prove a deployed shell has
  configured `VITE_INTAKE_SELECTION_BRIDGE_ORIGIN` or that a live Xplorer parent
  sends the expected message.

## Explicit exclusions and deployment hold

No hosted `xplorer-copilot` changes, shared/root manifests, secrets, network
configuration, deployment, or live message sender configuration are in scope.
Local source/test proof does not prove deployed origin/source configuration.
