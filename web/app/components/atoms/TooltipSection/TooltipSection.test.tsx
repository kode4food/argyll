import { render, screen } from "@testing-library/react";
import { IconInfo } from "@/utils/iconRegistry";
import TooltipSection from "./TooltipSection";

describe("TooltipSection", () => {
  test("renders title and children", () => {
    render(<TooltipSection title="Test Title">Test Content</TooltipSection>);
    expect(screen.getByText("Test Title:")).toBeInTheDocument();
    expect(screen.getByText("Test Content")).toBeInTheDocument();
  });

  test("renders without icon", () => {
    const { container } = render(
      <TooltipSection title="Title">Content</TooltipSection>
    );
    expect(container.querySelector("svg")).not.toBeInTheDocument();
  });

  test("renders with icon", () => {
    render(
      <TooltipSection title="Title" icon={<IconInfo data-testid="icon" />}>
        Content
      </TooltipSection>
    );
    expect(screen.getByTestId("icon")).toBeInTheDocument();
  });

  test("applies monospace styling", () => {
    const { container } = render(
      <TooltipSection title="Title" monospace>
        Content
      </TooltipSection>
    );
    const value = container.querySelector(".value");
    expect(value?.className).toContain("valueMonospace");
  });

  test("applies bold styling", () => {
    const { container } = render(
      <TooltipSection title="Title" bold>
        Content
      </TooltipSection>
    );
    const value = container.querySelector(".value");
    expect(value?.className).toContain("valueBold");
  });

  test("applies both monospace and bold styling", () => {
    const { container } = render(
      <TooltipSection title="Title" monospace bold>
        Content
      </TooltipSection>
    );
    const value = container.querySelector(".value");
    expect(value?.className).toContain("valueMonospace");
    expect(value?.className).toContain("valueBold");
  });

  test("defaults monospace to false", () => {
    const { container } = render(
      <TooltipSection title="Title">Content</TooltipSection>
    );
    const value = container.querySelector(".value");
    expect(value?.className).not.toContain("valueMonospace");
  });

  test("defaults bold to false", () => {
    const { container } = render(
      <TooltipSection title="Title">Content</TooltipSection>
    );
    const value = container.querySelector(".value");
    expect(value?.className).not.toContain("valueBold");
  });
});
