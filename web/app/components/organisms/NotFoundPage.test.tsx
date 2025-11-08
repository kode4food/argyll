import React from "react";
import { render, screen } from "@testing-library/react";
import NotFoundPage from "./NotFoundPage";

// Mock Next.js Link
jest.mock("next/link", () => {
  const MockLink = ({ children, href }: any) => <a href={href}>{children}</a>;
  MockLink.displayName = "MockLink";
  return MockLink;
});

describe("NotFoundPage", () => {
  test("renders 404 heading", () => {
    render(<NotFoundPage />);
    expect(screen.getByText("404 - Page Not Found")).toBeInTheDocument();
  });

  test("renders description message", () => {
    render(<NotFoundPage />);
    expect(
      screen.getByText(/The page you're looking for doesn't exist/)
    ).toBeInTheDocument();
  });

  test("renders back to overview link", () => {
    render(<NotFoundPage />);
    const link = screen.getByText("Back to Overview");
    expect(link).toBeInTheDocument();
    expect(link.closest("a")).toHaveAttribute("href", "/");
  });

  test("renders alert icon", () => {
    const { container } = render(<NotFoundPage />);
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });
});
