import { act, render, screen } from "@testing-library/react";
import { StepEditorProvider, useStepEditorContext } from "./StepEditorContext";
import { Step } from "@/app/api";

const mockStep: Step = {
  id: "s1",
  name: "Step 1",
  type: "script",
  attributes: {},
  script: { language: "lua", source: "" },
};

const renderSpy = jest.fn();
const mockMount = jest.fn();
jest.mock("@/app/components/organisms/StepEditor", () => {
  const { useEffect } = jest.requireActual("react");
  const Mock = ({ onUpdate }: any) => {
    renderSpy(onUpdate);
    useEffect(() => mockMount(), []);
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
    mockMount.mockClear();
  });

  it("remounts the editor when opened again while open", () => {
    render(
      <StepEditorProvider>
        <Consumer />
      </StepEditorProvider>
    );

    act(() => {
      screen.getByText("Open").click();
    });
    act(() => {
      screen.getByText("Open").click();
    });
    expect(mockMount).toHaveBeenCalledTimes(2);
  });

  it("opens and closes editor with active step", () => {
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
