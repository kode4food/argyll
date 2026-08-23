import { Handling, HTTPMethod, StepType } from "@/app/api";
import { getValidationError, parseFlowGoals } from "./stepValidationUtils";
import { Attribute } from "./stepEditorTypes";

const BASE_ARGS = {
  isCreateMode: false,
  stepId: "step-1",
  attributes: [] as Attribute[],
  stepType: "sync" as StepType,
  script: "",
  endpoint: "http://example.com",
  httpMethod: "POST" as HTTPMethod,
  httpTimeout: 1000,
  flowGoals: "",
  handling: "standard" as Handling,
  compensateEndpoint: "",
};

describe("stepValidationUtils", () => {
  test("parses flow goals", () => {
    expect(parseFlowGoals("step-a, step-b\n\nstep-c")).toEqual([
      "step-a",
      "step-b",
      "step-c",
    ]);
  });

  test("requires step id in create mode", () => {
    expect(
      getValidationError({
        ...BASE_ARGS,
        isCreateMode: true,
        stepId: " ",
      })
    ).toEqual({ key: "stepEditor.stepIdRequired" });
  });

  test("requires flow goals for flow steps", () => {
    expect(
      getValidationError({
        ...BASE_ARGS,
        stepType: "flow",
        flowGoals: " ",
      })
    ).toEqual({ key: "stepEditor.flowGoalsRequired" });
  });

  test("requires script for script steps", () => {
    expect(
      getValidationError({
        ...BASE_ARGS,
        stepType: "script",
        script: " ",
      })
    ).toEqual({ key: "stepEditor.scriptRequired" });
  });

  test("requires endpoint for http steps", () => {
    expect(
      getValidationError({
        ...BASE_ARGS,
        endpoint: " ",
      })
    ).toEqual({ key: "stepEditor.endpointRequired" });
  });

  test("requires GET params", () => {
    expect(
      getValidationError({
        ...BASE_ARGS,
        httpMethod: "GET",
        endpoint: "http://example.com/{account_id}",
        attributes: [
          {
            id: "attr-1",
            name: "customer_id",
            role: "required",
            dataType: "string",
          },
        ],
      })
    ).toEqual({
      key: "stepEditor.getEndpointParamRequired",
      vars: { name: "customer_id" },
    });
  });

  test("uses mapped GET params", () => {
    expect(
      getValidationError({
        ...BASE_ARGS,
        httpMethod: "GET",
        endpoint: "http://example.com/{customer_id}",
        attributes: [
          {
            id: "attr-1",
            name: "input",
            role: "required",
            dataType: "string",
            mappingName: "customer_id",
          },
        ],
      })
    ).toBeNull();
  });

  test("requires positive timeout for http steps", () => {
    expect(
      getValidationError({
        ...BASE_ARGS,
        httpTimeout: 0,
      })
    ).toEqual({ key: "stepEditor.timeoutPositive" });
  });

  test("requires an endpoint for compensated handling", () => {
    expect(
      getValidationError({
        ...BASE_ARGS,
        handling: "compensated",
      })
    ).toEqual({ key: "stepEditor.compensateEndpointRequired" });
  });

  test("rejects compensation with standard handling", () => {
    expect(
      getValidationError({
        ...BASE_ARGS,
        attributes: [
          {
            id: "attr-1",
            name: "request",
            role: "required",
            dataType: "string",
            compensated: true,
          },
        ],
      })
    ).toEqual({
      key: "stepEditor.compensatedAttributeHandlingRequired",
    });
  });

  test("rejects duplicate compensated inner names", () => {
    expect(
      getValidationError({
        ...BASE_ARGS,
        handling: "compensated",
        compensateEndpoint: "http://example.com/undo",
        attributes: [
          {
            id: "attr-1",
            name: "request",
            role: "required",
            dataType: "string",
            mappingName: "value",
            compensated: true,
          },
          {
            id: "attr-2",
            name: "result",
            role: "output",
            dataType: "string",
            mappingName: "value",
            compensated: true,
          },
        ],
      })
    ).toEqual({
      key: "stepEditor.compensatedAttributeConflict",
      vars: { name: "value" },
    });
  });
});
