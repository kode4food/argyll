import { renderHook } from "@testing-library/react";
import { useWorkflowFromUrl } from "./useWorkflowFromUrl";
import { useParams, usePathname } from "next/navigation";
import { useSelectWorkflow } from "../store/workflowStore";

jest.mock("next/navigation");
jest.mock("../store/workflowStore");

describe("useWorkflowFromUrl", () => {
  const mockUseParams = useParams as jest.MockedFunction<typeof useParams>;
  const mockUsePathname = usePathname as jest.MockedFunction<
    typeof usePathname
  >;
  const mockSelectWorkflow = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
    (useSelectWorkflow as jest.Mock).mockReturnValue(mockSelectWorkflow);
  });

  test("selects workflow from URL params", () => {
    mockUseParams.mockReturnValue({ workflowId: "wf-123" });
    mockUsePathname.mockReturnValue("/workflow/wf-123");

    const { result } = renderHook(() => useWorkflowFromUrl());

    expect(mockSelectWorkflow).toHaveBeenCalledWith("wf-123");
    expect(result.current).toBe("wf-123");
  });

  test("selects null workflow on home page", () => {
    mockUseParams.mockReturnValue({});
    mockUsePathname.mockReturnValue("/");

    renderHook(() => useWorkflowFromUrl());

    expect(mockSelectWorkflow).toHaveBeenCalledWith(null);
  });

  test("returns null when no workflow in URL", () => {
    mockUseParams.mockReturnValue({});
    mockUsePathname.mockReturnValue("/");

    const { result } = renderHook(() => useWorkflowFromUrl());

    expect(result.current).toBeNull();
  });

  test("does not select workflow for non-workflow paths", () => {
    mockUseParams.mockReturnValue({});
    mockUsePathname.mockReturnValue("/some-other-page");

    renderHook(() => useWorkflowFromUrl());

    expect(mockSelectWorkflow).not.toHaveBeenCalled();
  });

  test("updates when workflow ID changes", () => {
    mockUseParams.mockReturnValue({ workflowId: "wf-123" });
    mockUsePathname.mockReturnValue("/workflow/wf-123");

    const { rerender } = renderHook(() => useWorkflowFromUrl());

    expect(mockSelectWorkflow).toHaveBeenCalledWith("wf-123");

    mockUseParams.mockReturnValue({ workflowId: "wf-456" });
    mockUsePathname.mockReturnValue("/workflow/wf-456");

    rerender();

    expect(mockSelectWorkflow).toHaveBeenCalledWith("wf-456");
  });
});
