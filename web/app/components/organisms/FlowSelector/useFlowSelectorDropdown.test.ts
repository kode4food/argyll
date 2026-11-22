import React from "react";
import { renderHook } from "@testing-library/react";
import { useFlowSelectorDropdown } from "./useFlowSelectorDropdown";

const scrollEvent = (scrollTop: number) =>
  ({
    currentTarget: { scrollTop, clientHeight: 100, scrollHeight: 300 },
  }) as unknown as React.UIEvent<HTMLDivElement>;

const renderDropdown = (flowsHasMore: boolean) => {
  const loadMoreFlows = jest.fn().mockResolvedValue(undefined);
  const setScrollTop = jest.fn();
  const { result } = renderHook(() =>
    useFlowSelectorDropdown({
      showDropdown: true,
      selectedIndex: 0,
      selectedFlow: null,
      filteredFlows: [],
      flows: [],
      setScrollTop,
      flowsHasMore,
      flowsLoading: false,
      loadMoreFlows,
    })
  );
  return { result, loadMoreFlows, setScrollTop };
};

describe("useFlowSelectorDropdown", () => {
  test("loads more flows near the bottom", () => {
    const { result, loadMoreFlows, setScrollTop } = renderDropdown(true);
    result.current.handleDropdownScroll(scrollEvent(0));
    expect(loadMoreFlows).not.toHaveBeenCalled();
    result.current.handleDropdownScroll(scrollEvent(190));
    expect(setScrollTop).toHaveBeenCalledWith(190);
    expect(loadMoreFlows).toHaveBeenCalledTimes(1);
  });

  test("skips loading when no more flows", () => {
    const { result, loadMoreFlows } = renderDropdown(false);
    result.current.handleDropdownScroll(scrollEvent(190));
    expect(loadMoreFlows).not.toHaveBeenCalled();
  });
});
