import { FlowStatus } from "@/app/api";
import {
  SelectableFlow,
  filterFlowsBySearch,
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
});
