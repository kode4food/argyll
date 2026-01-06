import { renderHook } from "@testing-library/react";
import { useAttributeStatusBadge } from "./useAttributeDisplay";

describe("useAttributeStatusBadge", () => {
  it("returns a function", () => {
    const { result } = renderHook(() => useAttributeStatusBadge());
    expect(typeof result.current).toBe("function");
  });

  it("renders required satisfied status", () => {
    const { result } = renderHook(() => useAttributeStatusBadge());
    const badge = result.current("required", { isSatisfied: true });

    expect(badge).not.toBeNull();
    expect(badge?.props.className).toContain("satisfied");
  });

  it("renders required pending status", () => {
    const { result } = renderHook(() => useAttributeStatusBadge());
    const badge = result.current("required", { isSatisfied: false });

    expect(badge).not.toBeNull();
    expect(badge?.props.className).toContain("pending");
  });

  it("renders required failed status", () => {
    const { result } = renderHook(() => useAttributeStatusBadge());
    const badge = result.current("required", {
      isSatisfied: false,
      executionStatus: "failed",
    });

    expect(badge).not.toBeNull();
    expect(badge?.props.className).toContain("failed");
  });

  it("renders optional skipped status", () => {
    const { result } = renderHook(() => useAttributeStatusBadge());
    const badge = result.current("optional", {
      isSatisfied: false,
      executionStatus: "skipped",
    });

    expect(badge).not.toBeNull();
    expect(badge?.props.className).toContain("skipped");
  });

  it("renders optional satisfied status", () => {
    const { result } = renderHook(() => useAttributeStatusBadge());
    const badge = result.current("optional", {
      isSatisfied: false,
      executionStatus: "active",
      isProvidedByUpstream: true,
    });

    expect(badge).not.toBeNull();
    expect(badge?.props.className).toContain("satisfied");
  });

  it("renders optional defaulted status", () => {
    const { result } = renderHook(() => useAttributeStatusBadge());
    const badge = result.current("optional", {
      isSatisfied: false,
      executionStatus: "active",
      wasDefaulted: true,
    });

    expect(badge).not.toBeNull();
    expect(badge?.props.className).toContain("defaulted");
  });

  it("renders optional pending status", () => {
    const { result } = renderHook(() => useAttributeStatusBadge());
    const badge = result.current("optional", {
      isSatisfied: false,
      executionStatus: "active",
    });

    expect(badge).not.toBeNull();
    expect(badge?.props.className).toContain("pending");
  });

  it("renders output winner status", () => {
    const { result } = renderHook(() => useAttributeStatusBadge());
    const badge = result.current("output", {
      isSatisfied: false,
      executionStatus: "completed",
      isWinner: true,
    });

    expect(badge).not.toBeNull();
    expect(badge?.props.className).toContain("satisfied");
  });

  it("renders output non-winner status", () => {
    const { result } = renderHook(() => useAttributeStatusBadge());
    const badge = result.current("output", {
      isSatisfied: false,
      executionStatus: "completed",
      isWinner: false,
    });

    expect(badge).not.toBeNull();
    expect(badge?.props.className).toContain("not-winner");
  });

  it("renders output failed status", () => {
    const { result } = renderHook(() => useAttributeStatusBadge());
    const badge = result.current("output", {
      isSatisfied: false,
      executionStatus: "failed",
    });

    expect(badge).not.toBeNull();
    expect(badge?.props.className).toContain("skipped");
  });

  it("renders output active status", () => {
    const { result } = renderHook(() => useAttributeStatusBadge());
    const badge = result.current("output", {
      isSatisfied: false,
      executionStatus: "active",
    });

    expect(badge).not.toBeNull();
    expect(badge?.props.className).toContain("pending");
  });

  it("renders output placeholder when no status", () => {
    const { result } = renderHook(() => useAttributeStatusBadge());
    const badge = result.current("output", { isSatisfied: false });

    expect(badge).not.toBeNull();
    expect(badge?.props.className).toContain("placeholder");
  });

  it("memoizes the returned function", () => {
    const { result, rerender } = renderHook(() => useAttributeStatusBadge());
    const firstFunction = result.current;

    rerender();

    expect(result.current).toBe(firstFunction);
  });
});
