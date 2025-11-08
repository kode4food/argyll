import React from "react";
import { render, screen } from "@testing-library/react";
import EmptyState from "./EmptyState";
import { CheckCircle } from "lucide-react";

describe("EmptyState", () => {
  test("renders with title and description", () => {
    render(
      <EmptyState
        title="No Items Found"
        description="There are no items to display"
      />
    );
    expect(screen.getByText("No Items Found")).toBeInTheDocument();
    expect(
      screen.getByText("There are no items to display")
    ).toBeInTheDocument();
  });

  test("renders with default icon", () => {
    const { container } = render(
      <EmptyState title="Test" description="Test description" />
    );
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });

  test("renders with custom icon", () => {
    render(
      <EmptyState
        icon={<CheckCircle data-testid="custom-icon" />}
        title="Success"
        description="Operation completed"
      />
    );
    expect(screen.getByTestId("custom-icon")).toBeInTheDocument();
  });

  test("renders with action button", () => {
    render(
      <EmptyState
        title="No Data"
        description="Get started"
        action={<button>Create New</button>}
      />
    );
    expect(
      screen.getByRole("button", { name: "Create New" })
    ).toBeInTheDocument();
  });

  test("renders without action", () => {
    const { container } = render(
      <EmptyState title="Empty" description="Nothing here" />
    );
    expect(container.querySelector("button")).not.toBeInTheDocument();
  });

  test("applies custom className", () => {
    const { container } = render(
      <EmptyState
        title="Test"
        description="Test"
        className="custom-empty-state"
      />
    );
    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper.className).toContain("custom-empty-state");
  });

  test("renders title with correct styling", () => {
    render(<EmptyState title="My Title" description="Description" />);
    const title = screen.getByText("My Title");
    expect(title.tagName).toBe("H3");
    expect(title.className).toContain("text-xl");
    expect(title.className).toContain("font-medium");
  });

  test("renders description with correct styling", () => {
    render(<EmptyState title="Title" description="My description" />);
    const description = screen.getByText("My description");
    expect(description.tagName).toBe("P");
    expect(description.className).toContain("max-w-sm");
  });
});
