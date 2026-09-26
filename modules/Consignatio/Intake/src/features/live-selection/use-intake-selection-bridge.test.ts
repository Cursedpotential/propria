import { describe, expect, it } from "vitest";
import {
  applySelectionBridgeMessage,
  createSelectionBridgeTrust,
  getTrustedSelectionBridgeMessage,
  type BridgeSelectionState,
} from "./use-intake-selection-bridge";

const source = {} as MessageEventSource;
const otherSource = {} as MessageEventSource;
const origin = "https://intake.example.test";
const initial: BridgeSelectionState = {
  connected: false,
  currentPath: "",
  count: 0,
  selection: [],
};

const selectionPayload = {
  source: "xplorer-intake-bridge",
  type: "selection" as const,
  currentPath: "/case-files",
  count: 1,
  selection: [
    { name: "receipt.pdf", path: "/case-files/receipt.pdf", file_type: "pdf" },
  ],
};

function event(data: unknown, eventOrigin = origin, eventSource = source) {
  return { data, origin: eventOrigin, source: eventSource };
}

describe("Intake selection bridge trust boundary", () => {
  it("applies a structurally valid selection only from the configured origin and parent source", () => {
    const trust = createSelectionBridgeTrust(origin, source);
    const message = getTrustedSelectionBridgeMessage(event(selectionPayload), trust);

    expect(message).toEqual({
      type: "selection",
      currentPath: "/case-files",
      count: 1,
      selection: selectionPayload.selection,
    });
    expect(message).not.toBeNull();
    expect(applySelectionBridgeMessage(initial, message!)).toEqual({
      connected: true,
      currentPath: "/case-files",
      count: 1,
      selection: selectionPayload.selection,
    });
  });

  it("rejects a foreign origin without selection state side effects", () => {
    const message = getTrustedSelectionBridgeMessage(
      event(selectionPayload, "https://foreign.example.test"),
      createSelectionBridgeTrust(origin, source),
    );

    expect(message).toBeNull();
    expect(initial).toEqual({ connected: false, currentPath: "", count: 0, selection: [] });
  });

  it("rejects the wrong window source without selection state side effects", () => {
    const message = getTrustedSelectionBridgeMessage(
      event(selectionPayload, origin, otherSource),
      createSelectionBridgeTrust(origin, source),
    );

    expect(message).toBeNull();
    expect(initial).toEqual({ connected: false, currentPath: "", count: 0, selection: [] });
  });

  it("rejects malformed payloads without selection state side effects", () => {
    const malformed = {
      ...selectionPayload,
      selection: [{ name: "receipt.pdf", path: "/case-files/receipt.pdf" }],
    };
    const message = getTrustedSelectionBridgeMessage(
      event(malformed),
      createSelectionBridgeTrust(origin, source),
    );

    expect(message).toBeNull();
    expect(initial).toEqual({ connected: false, currentPath: "", count: 0, selection: [] });
  });

  it("fails closed when trusted origin or embedded parent source is absent", () => {
    expect(createSelectionBridgeTrust(undefined, source)).toBeNull();
    expect(createSelectionBridgeTrust(origin, null)).toBeNull();
    expect(
      getTrustedSelectionBridgeMessage(event(selectionPayload), null),
    ).toBeNull();
    expect(initial).toEqual({ connected: false, currentPath: "", count: 0, selection: [] });
  });
});
