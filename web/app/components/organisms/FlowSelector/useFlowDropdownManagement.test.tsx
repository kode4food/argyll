import { act, renderHook } from "@testing-library/react";
import { FlowContext } from "@/app/api";
import { useFlowDropdownManagement } from "./useFlowDropdownManagement";

const pushMock = jest.fn();

jest.mock("next/navigation", () => ({
  useRouter: () => ({
    push: pushMock,
  }),
}));

jest.mock("@/app/hooks/useEscapeKey", () => ({
  useEscapeKey: jest.fn(),
}));

const flows: FlowContext[] = [
  {
    id: "Overview",
    status: "pending",
    state: {},
    started_at: "2024-01-01T00:00:00Z",
  },
  {
    id: "flow-1",
    status: "active",
    state: {},
    started_at: "2024-01-02T00:00:00Z",
  },
];

describe("useFlowDropdownManagement", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    pushMock.mockReset();
  });

  it("filters flows based on sanitized search input", () => {
    const { result } = renderHook(() => useFlowDropdownManagement(flows, null));

    act(() => {
      result.current.handleSearchChange({
        target: { value: "FLOW 1" },
      } as React.ChangeEvent<HTMLInputElement>);
    });

    expect(result.current.filteredFlows).toHaveLength(1);
    expect(result.current.filteredFlows[0].id).toBe("flow-1");
    expect(result.current.selectedIndex).toBe(-1);
  });

  it("navigates with keyboard controls", () => {
    const { result } = renderHook(() => useFlowDropdownManagement(flows, null));

    act(() => {
      result.current.setShowDropdown(true);
    });

    act(() => {
      result.current.handleKeyDown({
        key: "ArrowDown",
        preventDefault: jest.fn(),
      } as any);
      result.current.handleKeyDown({
        key: "ArrowDown",
        preventDefault: jest.fn(),
      } as any);
    });

    expect(result.current.selectedIndex).toBe(1);

    act(() => {
      result.current.handleKeyDown({
        key: "Enter",
        preventDefault: jest.fn(),
      } as any);
    });

    expect(pushMock).toHaveBeenCalledWith("/flow/flow-1");
  });

  it("tabs to overview when none selected", () => {
    const { result } = renderHook(() => useFlowDropdownManagement(flows, null));

    act(() => {
      result.current.setShowDropdown(true);
    });

    act(() => {
      result.current.handleKeyDown({
        key: "Tab",
        preventDefault: jest.fn(),
      } as any);
    });

    expect(result.current.selectedIndex).toBe(0);

    act(() => {
      result.current.handleKeyDown({
        key: "Enter",
        preventDefault: jest.fn(),
      } as any);
    });

    expect(pushMock).toHaveBeenCalledWith("/");
  });

  it("resets state when closing dropdown", () => {
    const { result } = renderHook(() => useFlowDropdownManagement(flows, null));

    act(() => {
      result.current.setShowDropdown(true);
      result.current.handleSearchChange({
        target: { value: "flow" },
      } as React.ChangeEvent<HTMLInputElement>);
      result.current.handleKeyDown({
        key: "ArrowDown",
        preventDefault: jest.fn(),
      } as any);
    });

    act(() => {
      result.current.closeDropdown();
    });

    expect(result.current.showDropdown).toBe(false);
    expect(result.current.searchTerm).toBe("");
    expect(result.current.selectedIndex).toBe(-1);
  });
});
