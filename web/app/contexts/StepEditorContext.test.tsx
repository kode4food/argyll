import React from "react";
import { act, render, screen } from "@testing-library/react";
import { StepEditorProvider, useStepEditorContext } from "./StepEditorContext";
import { Step } from "../api";

const mockStep: Step = {
  id: "s1",
  name: "Step 1",
  type: "script",
  attributes: {},
  script: { language: "lua", script: "" },
};

const renderSpy = jest.fn();
jest.mock("../components/organisms/StepEditor", () => {
  const Mock = ({ onUpdate }: any) => {
    renderSpy(onUpdate);
    return (
      <div data-testid="editor">
        <button onClick={() => onUpdate(mockStep)}>Update</button>
      </div>
    );
  };
  return { __esModule: true, default: Mock };
});

const Consumer = () => {
  const { openEditor, closeEditor, isOpen, activeStep } =
    useStepEditorContext();
  return (
    <div>
      <button
        onClick={() =>
          openEditor({
            step: mockStep,
            onUpdate: jest.fn(),
          })
        }
      >
        Open
      </button>
      <button onClick={closeEditor}>Close</button>
      <span data-testid="is-open">{isOpen ? "yes" : "no"}</span>
      <span data-testid="active-step">{activeStep?.id || "none"}</span>
    </div>
  );
};

describe("StepEditorContext", () => {
  beforeEach(() => {
    renderSpy.mockClear();
  });

  it("opens, updates, and closes editor, exposing active step", () => {
    render(
      <StepEditorProvider>
        <Consumer />
      </StepEditorProvider>
    );

    expect(screen.getByTestId("is-open").textContent).toBe("no");
    expect(screen.getByTestId("active-step").textContent).toBe("none");

    act(() => {
      screen.getByText("Open").click();
    });

    expect(screen.getByTestId("is-open").textContent).toBe("yes");
    expect(screen.getByTestId("active-step").textContent).toBe("s1");
    expect(screen.getByTestId("editor")).toBeInTheDocument();

    // Trigger update from the mock editor
    act(() => {
      screen.getByText("Update").click();
    });
    expect(renderSpy).toHaveBeenCalledWith(expect.any(Function));

    act(() => {
      screen.getByText("Close").click();
    });
    expect(screen.getByTestId("is-open").textContent).toBe("no");
  });
});
