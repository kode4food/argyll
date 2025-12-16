import { FlowStatus } from "@/app/api";
import {
  SelectableFlow,
  createEventKey,
  extractFlowIdFromEvent,
  filterFlowsBySearch,
  flowExists,
  mapFlowStatusToProgressStatus,
} from "./flowSelectorUtils";

describe("flowSelectorUtils", () => {
  describe("mapFlowStatusToProgressStatus", () => {
    const cases: [FlowStatus, string][] = [
      ["pending", "pending"],
      ["active", "active"],
      ["completed", "completed"],
      ["failed", "failed"],
      ["stopped", "pending"],
    ];

    it.each(cases)("maps %s to %s", (status, expected) => {
      expect(mapFlowStatusToProgressStatus(status)).toBe(expected);
    });
  });

  it("filters flows with sanitized search term", () => {
    const flows: SelectableFlow[] = [
      { id: "demo-flow", status: "pending" },
      { id: "another_flow", status: "completed" },
    ];

    expect(filterFlowsBySearch(flows, "DEMO FLOW")).toEqual([flows[0]]);
    expect(filterFlowsBySearch(flows, "flow")).toEqual(flows);
    expect(filterFlowsBySearch(flows, "missing")).toEqual([]);
  });

  it("creates and parses event keys", () => {
    const key = createEventKey(["flow", "demo", "event"], 3);
    expect(key).toBe("flow:demo:event:3");
    expect(extractFlowIdFromEvent(["flow", "demo", "event"])).toBe("demo");
    expect(extractFlowIdFromEvent(["flow"])).toBeNull();
  });

  it("checks if a flow exists", () => {
    const flows: SelectableFlow[] = [
      { id: "flow-1", status: "pending" },
      { id: "flow-2", status: "failed" },
    ];

    expect(flowExists(flows, "flow-1")).toBe(true);
    expect(flowExists(flows, "flow-3")).toBe(false);
  });
});
