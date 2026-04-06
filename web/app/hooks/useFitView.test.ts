import { renderHook, act } from "@testing-library/react";
import { useFitView } from "./useFitView";

const mockGetNodes = jest.fn();
const mockGetNodesBounds = jest.fn();
const mockSetViewport = jest.fn();

jest.mock("@xyflow/react", () => ({
  useReactFlow: () => ({
    getNodes: mockGetNodes,
    getNodesBounds: mockGetNodesBounds,
    setViewport: mockSetViewport,
  }),
  getViewportForBounds: (_bounds: any, width: number, height: number) => ({
    x: width / 2 - 100,
    y: height / 2 - 50,
    zoom: 1,
  }),
}));

const containerRef = { current: null as HTMLDivElement | null };
const headerRef = { current: null as HTMLDivElement | null };
const panelRef = { current: null as HTMLDivElement | null };

jest.mock("@/app/contexts/UIContext", () => ({
  useUI: () => ({
    diagramContainerRef: containerRef,
    headerRef,
    panelRef,
    focusedPreviewAttribute: null,
    setFocusedPreviewAttribute: jest.fn(),
    previewPlan: null,
    setPreviewPlan: jest.fn(),
    goalSteps: [],
    toggleGoalStep: jest.fn(),
    setGoalSteps: jest.fn(),
    updatePreviewPlan: jest.fn(),
    clearPreviewPlan: jest.fn(),
  }),
}));

const makeContainer = (w: number, h: number): HTMLDivElement =>
  ({ clientWidth: w, clientHeight: h }) as unknown as HTMLDivElement;

const makeHeader = (height: number): HTMLDivElement =>
  ({ offsetHeight: height }) as unknown as HTMLDivElement;

const makePanel = (width: number): HTMLDivElement =>
  ({ offsetWidth: width }) as unknown as HTMLDivElement;

describe("useFitView", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGetNodes.mockReturnValue([{ id: "n1" }]);
    mockGetNodesBounds.mockReturnValue({ x: 0, y: 0, width: 200, height: 100 });
    containerRef.current = null;
    headerRef.current = null;
    panelRef.current = null;
  });

  test("does nothing when nodes are empty", () => {
    mockGetNodes.mockReturnValue([]);
    containerRef.current = makeContainer(1000, 800);

    const { result } = renderHook(() => useFitView());
    act(() => result.current());

    expect(mockSetViewport).not.toHaveBeenCalled();
  });

  test("does nothing when container ref is null", () => {
    containerRef.current = null;

    const { result } = renderHook(() => useFitView());
    act(() => result.current());

    expect(mockSetViewport).not.toHaveBeenCalled();
  });

  test("offsets y by header height", () => {
    containerRef.current = makeContainer(1000, 800);
    headerRef.current = makeHeader(60);
    panelRef.current = null;

    const { result } = renderHook(() => useFitView());
    act(() => result.current());

    expect(mockSetViewport).toHaveBeenCalled();
    const call = mockSetViewport.mock.calls[0][0];
    // getViewportForBounds called with visibleHeight=740, returns y=(740/2-50)=320
    // then offset by headerHeight 60 → 380
    expect(call.y).toBe(320 + 60);
    expect(call.zoom).toBe(1);
  });

  test("offsets x by panel width when panel ref is set", () => {
    containerRef.current = makeContainer(1000, 800);
    headerRef.current = null;
    panelRef.current = makePanel(260);

    const { result } = renderHook(() => useFitView());
    act(() => result.current());

    expect(mockSetViewport).toHaveBeenCalled();
    const call = mockSetViewport.mock.calls[0][0];
    // getViewportForBounds with visibleWidth=740, returns x=(740/2-100)=270, offset by 260 → 530
    expect(call.x).toBe(270 + 260);
  });

  test("does not offset x when panel ref is null", () => {
    containerRef.current = makeContainer(1000, 800);
    headerRef.current = null;
    panelRef.current = null;

    const { result } = renderHook(() => useFitView());
    act(() => result.current());

    expect(mockSetViewport).toHaveBeenCalled();
    const call = mockSetViewport.mock.calls[0][0];
    // Full width used: getViewportForBounds with visibleWidth=1000, x=(1000/2-100)=400
    expect(call.x).toBe(400);
  });
});
