import React from "react";
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import OverviewDiagram from "./OverviewDiagram";
import { UIProvider } from "@/app/contexts/UIContext";
import { FlowSessionProvider } from "@/app/contexts/FlowSessionContext";

const loadStepsMock = jest.fn().mockResolvedValue(undefined);
const loadFlowsMock = jest.fn().mockResolvedValue(undefined);
const flowStoreState = {
  selectFlow: jest.fn(),
  stepHealth: {},
  upsertStep: jest.fn(),
};

jest.mock("@/app/store/flowStore", () => {
  const api = jest.requireActual("@/app/store/flowStore");
  return {
    ...api,
    useSelectedFlow: jest.fn(() => null),
    useFlowStore: jest.fn((selector?: (state: any) => unknown) => {
      return selector ? selector(flowStoreState) : flowStoreState;
    }),
    useLoadFlows: jest.fn(() => loadFlowsMock),
    useLoadSteps: jest.fn(() => loadStepsMock),
    useSteps: jest.fn(() => [
      {
        id: "s1",
        name: "Step 1",
        type: "script",
        attributes: {},
        script: { language: "lua", script: "" },
      },
    ]),
    useFlows: jest.fn(() => []),
    useUpdateFlowStatus: jest.fn(() => jest.fn()),
    useFlowData: jest.fn(() => null),
    useFlowLoading: jest.fn(() => false),
    useFlowNotFound: jest.fn(() => false),
    useExecutions: jest.fn(() => []),
    useResolvedAttributes: jest.fn(() => []),
    useFlowError: jest.fn(() => null),
  };
});

jest.mock("@/app/components/organisms/StepEditor", () => {
  const Mock = ({ onUpdate }: any) => {
    React.useEffect(() => {
      onUpdate({
        id: "s1",
        name: "Updated",
        type: "script",
        attributes: {},
        script: { language: "lua", script: "" },
      });
    }, [onUpdate]);
    return <div data-testid="step-editor">Editor</div>;
  };
  return { __esModule: true, default: Mock };
});

describe("OverviewDiagram integration", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    flowStoreState.selectFlow.mockClear();
    flowStoreState.upsertStep.mockClear();
  });

  it("opens editor and applies step updates", async () => {
    render(
      <UIProvider>
        <FlowSessionProvider>
          <OverviewDiagram />
        </FlowSessionProvider>
      </UIProvider>
    );

    const createBtn = screen.getByRole("button", { name: /Create New Step/i });
    act(() => {
      fireEvent.click(createBtn);
    });

    expect(await screen.findByTestId("step-editor")).toBeInTheDocument();
    await waitFor(() => expect(flowStoreState.upsertStep).toHaveBeenCalled());
  });
});
