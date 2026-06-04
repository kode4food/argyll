import { HTTPMethod, StepType } from "@/app/api";
import { getValidationError, parseFlowGoals } from "./stepValidationUtils";
import { Attribute } from "./stepEditorTypes";

const baseArgs = {
  isCreateMode: false,
  stepId: "step-1",
  attributes: [] as Attribute[],
  stepType: "sync" as StepType,
  script: "",
  endpoint: "http://example.com",
  httpMethod: "POST" as HTTPMethod,
  httpTimeout: 1000,
  flowGoals: "",
};

describe("stepValidationUtils", () => {
  test("parses comma and newline separated flow goals", () => {
    expect(parseFlowGoals("step-a, step-b\n\nstep-c")).toEqual([
      "step-a",
      "step-b",
      "step-c",
    ]);
  });

  test("requires step id in create mode", () => {
    expect(
      getValidationError({
        ...baseArgs,
        isCreateMode: true,
        stepId: " ",
      })
    ).toEqual({ key: "stepEditor.stepIdRequired" });
  });

  test("requires flow goals for flow steps", () => {
    expect(
      getValidationError({
        ...baseArgs,
        stepType: "flow",
        flowGoals: " ",
      })
    ).toEqual({ key: "stepEditor.flowGoalsRequired" });
  });

  test("requires script for script steps", () => {
    expect(
      getValidationError({
        ...baseArgs,
        stepType: "script",
        script: " ",
      })
    ).toEqual({ key: "stepEditor.scriptRequired" });
  });

  test("requires endpoint for http steps", () => {
    expect(
      getValidationError({
        ...baseArgs,
        endpoint: " ",
      })
    ).toEqual({ key: "stepEditor.endpointRequired" });
  });

  test("requires GET endpoint params for required attributes", () => {
    expect(
      getValidationError({
        ...baseArgs,
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

  test("uses mapping name when validating GET endpoint params", () => {
    expect(
      getValidationError({
        ...baseArgs,
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
        ...baseArgs,
        httpTimeout: 0,
      })
    ).toEqual({ key: "stepEditor.timeoutPositive" });
  });
});
