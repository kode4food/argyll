import React from "react";
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import FlowDiagram from "./FlowDiagram";
import { UIProvider } from "../../contexts/UIContext";
import { FlowSessionProvider } from "../../contexts/FlowSessionContext";

jest.mock("./FlowDiagram/useFlowWebSocket", () => ({
  useFlowWebSocket: jest.fn(),
}));

const loadStepsMock = jest.fn().mockResolvedValue(undefined);
const loadFlowsMock = jest.fn().mockResolvedValue(undefined);

jest.mock("../../store/flowStore", () => {
  const api = jest.requireActual("../../store/flowStore");
  return {
    ...api,
    useSelectedFlow: jest.fn(() => null),
    useFlowStore: jest.fn(() => ({ selectFlow: jest.fn() })),
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
    useIsFlowMode: jest.fn(() => false),
    useExecutions: jest.fn(() => []),
    useResolvedAttributes: jest.fn(() => []),
    useFlowError: jest.fn(() => null),
  };
});

jest.mock("../organisms/StepEditor", () => {
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

describe("FlowDiagram integration", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("opens editor and reloads steps on create", async () => {
    render(
      <UIProvider>
        <FlowSessionProvider>
          <FlowDiagram />
        </FlowSessionProvider>
      </UIProvider>
    );

    const createBtn = screen.getByRole("button", { name: /Create New Step/i });
    act(() => {
      fireEvent.click(createBtn);
    });

    expect(await screen.findByTestId("step-editor")).toBeInTheDocument();
    await waitFor(() => expect(loadStepsMock).toHaveBeenCalled());
  });
});
