import { renderHook } from "@testing-library/react";
import { useFlowFromUrl } from "./useFlowFromUrl";
import { useParams, useLocation } from "react-router-dom";
import { useSelectFlow } from "@/app/store/flowStore";

jest.mock("react-router-dom", () => ({
  ...jest.requireActual("react-router-dom"),
  useParams: jest.fn(),
  useLocation: jest.fn(),
}));
jest.mock("@/app/store/flowStore");

describe("useFlowFromUrl", () => {
  const mockUseParams = useParams as jest.MockedFunction<typeof useParams>;
  const mockUseLocation = useLocation as jest.MockedFunction<
    typeof useLocation
  >;
  const mockSelectFlow = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
    (useSelectFlow as jest.Mock).mockReturnValue(mockSelectFlow);
  });

  test("selects flow from URL params", () => {
    mockUseParams.mockReturnValue({ flowId: "wf-123" });
    mockUseLocation.mockReturnValue({ pathname: "/flow/wf-123" });

    const { result } = renderHook(() => useFlowFromUrl());

    expect(mockSelectFlow).toHaveBeenCalledWith("wf-123");
    expect(result.current).toBe("wf-123");
  });

  test("selects null flow on home page", () => {
    mockUseParams.mockReturnValue({});
    mockUseLocation.mockReturnValue({ pathname: "/" });

    renderHook(() => useFlowFromUrl());

    expect(mockSelectFlow).toHaveBeenCalledWith(null);
  });

  test("returns null when no flow in URL", () => {
    mockUseParams.mockReturnValue({});
    mockUseLocation.mockReturnValue({ pathname: "/" });

    const { result } = renderHook(() => useFlowFromUrl());

    expect(result.current).toBeNull();
  });

  test("does not select flow for non-flow paths", () => {
    mockUseParams.mockReturnValue({});
    mockUseLocation.mockReturnValue({ pathname: "/some-other-page" });

    renderHook(() => useFlowFromUrl());

    expect(mockSelectFlow).not.toHaveBeenCalled();
  });

  test("updates when flow ID changes", () => {
    mockUseParams.mockReturnValue({ flowId: "wf-123" });
    mockUseLocation.mockReturnValue({ pathname: "/flow/wf-123" });

    const { rerender } = renderHook(() => useFlowFromUrl());

    expect(mockSelectFlow).toHaveBeenCalledWith("wf-123");

    mockUseParams.mockReturnValue({ flowId: "wf-456" });
    mockUseLocation.mockReturnValue({ pathname: "/flow/wf-456" });

    rerender();

    expect(mockSelectFlow).toHaveBeenCalledWith("wf-456");
  });
});
