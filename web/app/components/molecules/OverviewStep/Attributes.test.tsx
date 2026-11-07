import { render } from "@testing-library/react";
import Attributes from "./Attributes";
import { Step, AttributeRole } from "@/app/api";

describe("Attributes", () => {
  test("renders attributes with arg markers", () => {
    const step: Step = {
      id: "step-1",
      name: "Test Step",
      type: "service",
      attributes: {
        input1: { role: AttributeRole.Required },
        output1: { role: AttributeRole.Output },
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

  test("highlights focused attribute rows", () => {
    const step: Step = {
      id: "step-1",
      name: "Test Step",
      type: "service",
      attributes: {
        quantity: { role: AttributeRole.Required },
        total: { role: AttributeRole.Output },
      },
      http: {
        endpoint: "http://test",
        timeout: 5000,
      },
    };

    const { container } = render(
      <Attributes step={step} focusedAttributeName="quantity" />
    );

    const focusedRow = container.querySelector('[data-arg-name="quantity"]');
    expect(focusedRow?.className).toContain("argItemFocused");
    expect(focusedRow?.className).toContain("argItemFocusedInput");
  });

  test("uses outward pulse for focused outputs", () => {
    const step: Step = {
      id: "step-1",
      name: "Test Step",
      type: "service",
      attributes: {
        quantity: { role: AttributeRole.Required },
        total: { role: AttributeRole.Output },
      },
      http: {
        endpoint: "http://test",
        timeout: 5000,
      },
    };

    const { container } = render(
      <Attributes step={step} focusedAttributeName="total" />
    );

    const focusedRow = container.querySelector('[data-arg-name="total"]');
    expect(focusedRow?.className).toContain("argItemFocused");
    expect(focusedRow?.className).toContain("argItemFocusedOutput");
  });

  test("previews declared values", () => {
    const step: Step = {
      id: "step-1",
      name: "Test Step",
      type: "service",
      attributes: {
        input1: { role: AttributeRole.Required },
        opt1: { role: AttributeRole.Optional, optional: { default: "42" } },
        region: { role: AttributeRole.Const, const: { value: '"eu"' } },
        flow: { role: AttributeRole.Meta, meta: { key: "flow_id" } },
        total: { role: AttributeRole.Feed, feed: { id: "order-total" } },
      },
      http: {
        endpoint: "http://test",
        timeout: 5000,
      },
    };

    const { container } = render(<Attributes step={step} />);
    const preview = (name: string) =>
      container.querySelector(`[data-arg-name="${name}"] .argPreview`)
        ?.textContent;

    expect(preview("input1")).toBeUndefined();
    expect(preview("opt1")).toBe("42");
    expect(preview("region")).toBe('"eu"');
    expect(preview("flow")).toBe("flow_id");
    expect(preview("total")).toBe("order-total");
  });
});
