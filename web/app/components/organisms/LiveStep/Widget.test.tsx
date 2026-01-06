import React from "react";
import { render, fireEvent } from "@testing-library/react";
import Widget from "./Widget";
import type { Step, ExecutionResult } from "@/app/api";

jest.mock("@/app/components/molecules/LiveStep/Attributes", () => ({
  __esModule: true,
  default: ({ step }: any) => <div data-testid="step-args">{step.id}</div>,
}));

jest.mock("@/app/components/molecules/LiveStep/Footer", () => ({
  __esModule: true,
  default: ({ step }: any) => <div data-testid="step-footer">{step.id}</div>,
}));

describe("Widget", () => {
  const step: Step = {
    id: "step-1",
    name: "Test Step",
    type: "sync",
    attributes: {},
    http: {
      endpoint: "http://localhost:8080/test",
      timeout: 5000,
    },
  };

  test("calls onClick when clicked", () => {
    const onClick = jest.fn();
    const { container } = render(<Widget step={step} onClick={onClick} />);

    const widget = container.querySelector(".step-widget");
    fireEvent.click(widget!);

    expect(onClick).toHaveBeenCalledTimes(1);
  });

  test("renders execution props", () => {
    const execution: ExecutionResult = {
      step_id: "step-1",
      flow_id: "wf-1",
      status: "completed",
      inputs: {},
      started_at: "2024-01-01T00:00:00Z",
    };

    const { getByTestId } = render(
      <Widget step={step} execution={execution} />
    );

    expect(getByTestId("step-footer")).toBeInTheDocument();
  });
});
