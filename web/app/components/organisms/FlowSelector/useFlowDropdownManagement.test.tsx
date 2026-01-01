import { act, renderHook } from "@testing-library/react";
import { FlowContext } from "@/app/api";
import { useFlowDropdownManagement } from "./useFlowDropdownManagement";

const pushMock = jest.fn();

jest.mock("react-router-dom", () => ({
  ...jest.requireActual("react-router-dom"),
  useNavigate: () => pushMock,
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

  describe("arrow key navigation edge cases", () => {
    it("wraps ArrowUp from first item to last item", () => {
      const { result } = renderHook(() =>
        useFlowDropdownManagement(flows, null)
      );

      act(() => {
        result.current.setShowDropdown(true);
      });

      act(() => {
        result.current.handleKeyDown({
          key: "ArrowUp",
          preventDefault: jest.fn(),
        } as any);
      });

      expect(result.current.selectedIndex).toBe(flows.length - 1);
    });

    it("navigates up with ArrowUp when not at first item", () => {
      const { result } = renderHook(() =>
        useFlowDropdownManagement(flows, null)
      );

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

      act(() => {
        result.current.handleKeyDown({
          key: "ArrowUp",
          preventDefault: jest.fn(),
        } as any);
      });

      expect(result.current.selectedIndex).toBe(0);
    });

    it("wraps ArrowDown from last item to first item", () => {
      const { result } = renderHook(() =>
        useFlowDropdownManagement(flows, null)
      );

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
        result.current.handleKeyDown({
          key: "ArrowDown",
          preventDefault: jest.fn(),
        } as any);
      });

      expect(result.current.selectedIndex).toBe(0);
    });
  });

  describe("tab key navigation edge cases", () => {
    it("tabs when valid selection exists", () => {
      const { result } = renderHook(() =>
        useFlowDropdownManagement(flows, null)
      );

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

      const initialCallCount = pushMock.mock.calls.length;

      act(() => {
        result.current.handleKeyDown({
          key: "Tab",
          preventDefault: jest.fn(),
        } as any);
      });

      expect(pushMock.mock.calls.length).toBeGreaterThan(initialCallCount);
    });

    it("does not navigate on Tab without valid selection", () => {
      const { result } = renderHook(() =>
        useFlowDropdownManagement(flows, null)
      );

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
    });
  });

  describe("dropdown focus and scroll behavior", () => {
    it("ignores keyboard events when dropdown is closed", () => {
      const { result } = renderHook(() =>
        useFlowDropdownManagement(flows, null)
      );

      act(() => {
        result.current.setShowDropdown(false);
      });

      act(() => {
        result.current.handleKeyDown({
          key: "ArrowDown",
          preventDefault: jest.fn(),
        } as any);
      });

      expect(result.current.selectedIndex).toBe(-1);
    });

    it("navigates to Overview with 'Overview' flow id", () => {
      const { result } = renderHook(() =>
        useFlowDropdownManagement(flows, null)
      );

      act(() => {
        result.current.setShowDropdown(true);
      });

      act(() => {
        result.current.selectFlow("Overview");
      });

      expect(pushMock).toHaveBeenCalledWith("/");
    });

    it("scrolls selected item into view when selection changes", () => {
      const scrollIntoView = jest.fn();
      const mockElement = {
        scrollIntoView: scrollIntoView,
      };

      const { result } = renderHook(() =>
        useFlowDropdownManagement(flows, null)
      );

      const mockChildren = [null, mockElement] as any[];
      Object.defineProperty(result.current.dropdownRef, "current", {
        value: { children: mockChildren },
        writable: true,
      });

      act(() => {
        result.current.setShowDropdown(true);
      });

      act(() => {
        result.current.handleKeyDown({
          key: "ArrowDown",
          preventDefault: jest.fn(),
        } as any);
      });

      expect(scrollIntoView).toHaveBeenCalledWith({
        behavior: "smooth",
        block: "nearest",
      });
    });

    it("does not scroll when dropdownRef is null", () => {
      const { result } = renderHook(() =>
        useFlowDropdownManagement(flows, null)
      );

      act(() => {
        result.current.setShowDropdown(true);
        result.current.handleKeyDown({
          key: "ArrowDown",
          preventDefault: jest.fn(),
        } as any);
      });

      expect(result.current.dropdownRef.current).toBeNull();
    });

    it("does not scroll when selected element is not found", () => {
      const scrollIntoView = jest.fn();

      const { result } = renderHook(() =>
        useFlowDropdownManagement(flows, null)
      );

      Object.defineProperty(result.current.dropdownRef, "current", {
        value: { children: [] as any[] },
        writable: true,
      });

      act(() => {
        result.current.setShowDropdown(true);
        result.current.handleKeyDown({
          key: "ArrowDown",
          preventDefault: jest.fn(),
        } as any);
      });

      expect(scrollIntoView).not.toHaveBeenCalled();
    });
  });
});
