import { render } from "@testing-library/react";
import Attributes from "./Attributes";
import { Step, AttributeRole, AttributeType } from "@/app/api";

describe("Attributes", () => {
  test("renders attributes with arg markers", () => {
    const step: Step = {
      id: "step-1",
      name: "Test Step",
      type: "sync",
      attributes: {
        input1: { role: AttributeRole.Required, type: AttributeType.String },
        output1: { role: AttributeRole.Output, type: AttributeType.String },
      },
      http: {
        endpoint: "http://test",
        timeout: 5000,
      },
    };

    const { container } = render(<Attributes step={step} />);

    expect(
      container.querySelector('[data-arg-name="input1"]')
    ).toBeInTheDocument();
    expect(
      container.querySelector('[data-arg-name="output1"]')
    ).toBeInTheDocument();
  });
});
