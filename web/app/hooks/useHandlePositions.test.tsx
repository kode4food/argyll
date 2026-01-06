import React from "react";
import { renderHook } from "@testing-library/react";
import { useHandlePositions } from "./useHandlePositions";
import { AttributeRole, AttributeType } from "@/app/api";

describe("useHandlePositions", () => {
  it("initializes with empty handle positions", () => {
    const step = {
      id: "step-1",
      name: "Test",
      type: "sync" as const,
      attributes: {},
    };
    const ref = React.createRef<HTMLDivElement>();

    const { result } = renderHook(() => useHandlePositions(step as any, ref));

    expect(result.current.handlePositions.required).toEqual([]);
    expect(result.current.handlePositions.optional).toEqual([]);
    expect(result.current.handlePositions.output).toEqual([]);
    expect(result.current.allHandles).toEqual([]);
  });

  it("flattens all handles into allHandles array", () => {
    const step = {
      id: "step-1",
      name: "Test",
      type: "sync" as const,
      attributes: {
        req1: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
        opt1: {
          role: AttributeRole.Optional,
          type: AttributeType.String,
          description: "",
        },
        out1: {
          role: AttributeRole.Output,
          type: AttributeType.String,
          description: "",
        },
      },
    };

    // Create a mock div with child elements
    const mockDiv = document.createElement("div");
    const mockReq = document.createElement("div");
    const mockOpt = document.createElement("div");
    const mockOut = document.createElement("div");

    mockReq.setAttribute("data-arg-type", "required");
    mockReq.setAttribute("data-arg-name", "req1");
    mockReq.style.top = "10px";
    mockReq.style.height = "20px";

    mockOpt.setAttribute("data-arg-type", "optional");
    mockOpt.setAttribute("data-arg-name", "opt1");
    mockOpt.style.top = "50px";
    mockOpt.style.height = "20px";

    mockOut.setAttribute("data-arg-type", "output");
    mockOut.setAttribute("data-arg-name", "out1");
    mockOut.style.top = "100px";
    mockOut.style.height = "20px";

    mockDiv.appendChild(mockReq);
    mockDiv.appendChild(mockOpt);
    mockDiv.appendChild(mockOut);

    const ref = React.createRef<HTMLDivElement>();
    Object.defineProperty(ref, "current", {
      value: mockDiv,
      writable: false,
    });

    const { result } = renderHook(() => useHandlePositions(step as any, ref));

    expect(result.current.allHandles).toHaveLength(3);
    expect(result.current.allHandles.some((h) => h.argName === "req1")).toBe(
      true
    );
    expect(result.current.allHandles.some((h) => h.argName === "opt1")).toBe(
      true
    );
    expect(result.current.allHandles.some((h) => h.argName === "out1")).toBe(
      true
    );
  });

  it("returns empty arrays when ref is null", () => {
    const step = {
      id: "step-1",
      name: "Test",
      type: "sync" as const,
      attributes: {
        req1: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
      },
    };
    const ref = React.createRef<HTMLDivElement>();

    const { result } = renderHook(() => useHandlePositions(step as any, ref));

    expect(result.current.handlePositions.required).toEqual([]);
    expect(result.current.allHandles).toEqual([]);
  });

  it("handles step attributes change", () => {
    const ref = React.createRef<HTMLDivElement>();

    const step1 = {
      id: "step-1",
      name: "Test",
      type: "sync" as const,
      attributes: {
        req1: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
      },
    };

    const step2 = {
      id: "step-1",
      name: "Test",
      type: "sync" as const,
      attributes: {
        req1: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
        req2: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
      },
    };

    const { rerender } = renderHook(
      ({ step }) => useHandlePositions(step as any, ref),
      { initialProps: { step: step1 } }
    );

    // Verify hook doesn't throw when attributes change
    rerender({ step: step2 });

    // Should update without errors
    expect(true).toBe(true);
  });

  it("generates correct handle IDs", () => {
    const step = {
      id: "step-1",
      name: "Test",
      type: "sync" as const,
      attributes: {
        username: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
      },
    };

    const mockDiv = document.createElement("div");
    const mockInput = document.createElement("div");
    mockInput.setAttribute("data-arg-type", "required");
    mockInput.setAttribute("data-arg-name", "username");
    mockDiv.appendChild(mockInput);

    const ref = React.createRef<HTMLDivElement>();
    Object.defineProperty(ref, "current", { value: mockDiv });

    const { result } = renderHook(() => useHandlePositions(step as any, ref));

    const handle = result.current.handlePositions.required[0];
    expect(handle.id).toBe("input-required-username");
  });

  it("sets correct handle types", () => {
    const step = {
      id: "step-1",
      name: "Test",
      type: "sync" as const,
      attributes: {
        input1: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
        output1: {
          role: AttributeRole.Output,
          type: AttributeType.String,
          description: "",
        },
      },
    };

    const mockDiv = document.createElement("div");

    const mockInput = document.createElement("div");
    mockInput.setAttribute("data-arg-type", "required");
    mockInput.setAttribute("data-arg-name", "input1");
    mockDiv.appendChild(mockInput);

    const mockOutput = document.createElement("div");
    mockOutput.setAttribute("data-arg-type", "output");
    mockOutput.setAttribute("data-arg-name", "output1");
    mockDiv.appendChild(mockOutput);

    const ref = React.createRef<HTMLDivElement>();
    Object.defineProperty(ref, "current", { value: mockDiv });

    const { result } = renderHook(() => useHandlePositions(step as any, ref));

    const inputHandle = result.current.handlePositions.required[0];
    const outputHandle = result.current.handlePositions.output[0];

    expect(inputHandle.handleType).toBe("input");
    expect(outputHandle.handleType).toBe("output");
  });

  it("handles mixed found and missing DOM elements", () => {
    const step = {
      id: "step-1",
      name: "Test",
      type: "sync" as const,
      attributes: {
        req1: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
        req2: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
      },
    };

    const mockDiv = document.createElement("div");
    const mockReq1 = document.createElement("div");
    mockReq1.setAttribute("data-arg-type", "required");
    mockReq1.setAttribute("data-arg-name", "req1");
    mockReq1.style.top = "10px";
    mockReq1.style.height = "20px";
    mockDiv.appendChild(mockReq1);

    const ref = React.createRef<HTMLDivElement>();
    Object.defineProperty(ref, "current", { value: mockDiv });

    const { result } = renderHook(() => useHandlePositions(step as any, ref));

    expect(result.current.handlePositions.required).toHaveLength(1);
  });

  it("calculates correct top position with different element heights", () => {
    const step = {
      id: "step-1",
      name: "Test",
      type: "sync" as const,
      attributes: {
        arg: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
      },
    };

    const mockDiv = document.createElement("div");
    const mockElement = document.createElement("div");
    mockElement.setAttribute("data-arg-type", "required");
    mockElement.setAttribute("data-arg-name", "arg");
    mockDiv.appendChild(mockElement);

    Object.defineProperty(mockElement, "offsetTop", {
      value: 50,
      writable: true,
    });
    Object.defineProperty(mockElement, "offsetHeight", {
      value: 40,
      writable: true,
    });

    const ref = React.createRef<HTMLDivElement>();
    Object.defineProperty(ref, "current", { value: mockDiv });

    const { result } = renderHook(() => useHandlePositions(step as any, ref));

    const handle = result.current.handlePositions.required[0];
    expect(handle.top).toBe(70);
  });

  it("handles updates when attributes are added", () => {
    const ref = React.createRef<HTMLDivElement>();
    const mockDiv = document.createElement("div");

    const mockReq = document.createElement("div");
    mockReq.setAttribute("data-arg-type", "required");
    mockReq.setAttribute("data-arg-name", "req1");
    mockReq.style.top = "10px";
    mockReq.style.height = "20px";
    mockDiv.appendChild(mockReq);

    Object.defineProperty(ref, "current", { value: mockDiv });

    const step1 = {
      id: "step-1",
      name: "Test",
      type: "sync" as const,
      attributes: {
        req1: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
      },
    };

    const step2 = {
      id: "step-1",
      name: "Test",
      type: "sync" as const,
      attributes: {
        req1: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
        opt1: {
          role: AttributeRole.Optional,
          type: AttributeType.String,
          description: "",
        },
      },
    };

    const { result, rerender } = renderHook(
      ({ step }) => useHandlePositions(step as any, ref),
      { initialProps: { step: step1 } }
    );

    expect(result.current.allHandles).toHaveLength(1);

    const mockOpt = document.createElement("div");
    mockOpt.setAttribute("data-arg-type", "optional");
    mockOpt.setAttribute("data-arg-name", "opt1");
    mockOpt.style.top = "50px";
    mockOpt.style.height = "20px";
    mockDiv.appendChild(mockOpt);

    rerender({ step: step2 });

    expect(result.current.allHandles).toHaveLength(2);
  });

  it("preserves handle order across rerenders", () => {
    const step = {
      id: "step-1",
      name: "Test",
      type: "sync" as const,
      attributes: {
        req1: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
        req2: {
          role: AttributeRole.Required,
          type: AttributeType.String,
          description: "",
        },
      },
    };

    const mockDiv = document.createElement("div");
    const mockReq1 = document.createElement("div");
    mockReq1.setAttribute("data-arg-type", "required");
    mockReq1.setAttribute("data-arg-name", "req1");
    mockReq1.style.top = "10px";
    mockReq1.style.height = "20px";
    mockDiv.appendChild(mockReq1);

    const mockReq2 = document.createElement("div");
    mockReq2.setAttribute("data-arg-type", "required");
    mockReq2.setAttribute("data-arg-name", "req2");
    mockReq2.style.top = "50px";
    mockReq2.style.height = "20px";
    mockDiv.appendChild(mockReq2);

    const ref = React.createRef<HTMLDivElement>();
    Object.defineProperty(ref, "current", { value: mockDiv });

    const { result, rerender } = renderHook(
      ({ step }) => useHandlePositions(step as any, ref),
      { initialProps: { step } }
    );

    const firstRenderOrder = result.current.allHandles.map((h) => h.argName);

    rerender({ step });

    const secondRenderOrder = result.current.allHandles.map((h) => h.argName);

    expect(firstRenderOrder).toEqual(secondRenderOrder);
  });
});
