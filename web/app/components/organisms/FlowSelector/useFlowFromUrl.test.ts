import { renderHook } from "@testing-library/react";
import { useFlowFromUrl } from "./useFlowFromUrl";
import { useParams, usePathname } from "next/navigation";
import { useSelectFlow } from "@/app/store/flowStore";

jest.mock("next/navigation");
jest.mock("@/app/store/flowStore");

describe("useFlowFromUrl", () => {
  const mockUseParams = useParams as jest.MockedFunction<typeof useParams>;
  const mockUsePathname = usePathname as jest.MockedFunction<
    typeof usePathname
  >;
  const mockSelectFlow = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
    (useSelectFlow as jest.Mock).mockReturnValue(mockSelectFlow);
  });

  test("selects flow from URL params", () => {
    mockUseParams.mockReturnValue({ flowId: "wf-123" });
    mockUsePathname.mockReturnValue("/flow/wf-123");

    const { result } = renderHook(() => useFlowFromUrl());

    expect(mockSelectFlow).toHaveBeenCalledWith("wf-123");
    expect(result.current).toBe("wf-123");
  });

  test("selects null flow on home page", () => {
    mockUseParams.mockReturnValue({});
    mockUsePathname.mockReturnValue("/");

    renderHook(() => useFlowFromUrl());

    expect(mockSelectFlow).toHaveBeenCalledWith(null);
  });

  test("returns null when no flow in URL", () => {
    mockUseParams.mockReturnValue({});
    mockUsePathname.mockReturnValue("/");

    const { result } = renderHook(() => useFlowFromUrl());

    expect(result.current).toBeNull();
  });

  test("does not select flow for non-flow paths", () => {
    mockUseParams.mockReturnValue({});
    mockUsePathname.mockReturnValue("/some-other-page");

    renderHook(() => useFlowFromUrl());

    expect(mockSelectFlow).not.toHaveBeenCalled();
  });

  test("updates when flow ID changes", () => {
    mockUseParams.mockReturnValue({ flowId: "wf-123" });
    mockUsePathname.mockReturnValue("/flow/wf-123");

    const { rerender } = renderHook(() => useFlowFromUrl());

    expect(mockSelectFlow).toHaveBeenCalledWith("wf-123");

    mockUseParams.mockReturnValue({ flowId: "wf-456" });
    mockUsePathname.mockReturnValue("/flow/wf-456");

    rerender();

    expect(mockSelectFlow).toHaveBeenCalledWith("wf-456");
  });
});
