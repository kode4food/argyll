import React from "react";
import { render } from "@testing-library/react";
import Node from "./Node";
import { Step, AttributeRole, AttributeType } from "@/app/api";
import {
  DiagramSelectionProvider,
  DiagramSelectionContextValue,
} from "@/app/contexts/DiagramSelectionContext";
import { UIProvider } from "@/app/contexts/UIContext";

jest.mock("@/app/contexts/UIContext", () => ({
  UIProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  useUI: () => ({
    goalSteps: [],
    toggleGoalStep: jest.fn(),
    setGoalSteps: jest.fn(),
    disableEdit: false,
    diagramContainerRef: { current: null },
  }),
}));

jest.mock("./Widget", () => {
  return function MockWidget({ step }: any) {
    return (
      <div data-testid="step-widget" data-step-id={step.id}>
        {Object.entries(step.attributes || {}).map(
          ([name, spec]: [string, any]) => (
            <div
              key={name}
              data-arg-type={spec.role}
              data-arg-name={name}
              style={{ height: 20 }}
            />
          )
        )}
      </div>
    );
  };
});

jest.mock("@/app/components/atoms/InvisibleHandle", () => {
  return function MockInvisibleHandle({ id }: any) {
    return <div data-testid={`handle-${id}`} />;
  };
});

describe("Node", () => {
  type TestStepNodeData = {
    step: Step;
    selected: boolean;
  };

  const mockStep: Step = {
    id: "step-1",
    name: "Test Step",
    type: "sync",
    attributes: {
      input1: { role: AttributeRole.Required, type: AttributeType.String },
      output1: { role: AttributeRole.Output, type: AttributeType.String },
    },
    http: {
      endpoint: "http://localhost:8080/test",
      timeout: 5000,
    },
  };

  const defaultNodeData: TestStepNodeData = {
    step: mockStep,
    selected: false,
  };

  const defaultProps = {
    id: "node-1",
    type: "step",
    data: defaultNodeData,
    selected: false,
    isConnectable: true,
    zIndex: 0,
    xPos: 0,
    yPos: 0,
    positionAbsoluteX: 0,
    positionAbsoluteY: 0,
    dragging: false,
    draggable: true,
    selectable: true,
    deletable: false,
  };

  const renderWithProvider = (
    nodeOverrides: Partial<typeof defaultProps> = {},
    selectionOverrides: Partial<DiagramSelectionContextValue> = {}
  ) =>
    render(
      <UIProvider>
        <DiagramSelectionProvider
          value={{
            goalSteps: [],
            toggleGoalStep: jest.fn(),
            setGoalSteps: jest.fn(),
            ...selectionOverrides,
          }}
        >
          <Node {...defaultProps} {...nodeOverrides} />
        </DiagramSelectionProvider>
      </UIProvider>
    );

  test("renders handles for attributes", () => {
    const { getByTestId } = renderWithProvider();
    expect(getByTestId("handle-input-required-input1")).toBeInTheDocument();
    expect(getByTestId("handle-output-output1")).toBeInTheDocument();
  });
});
