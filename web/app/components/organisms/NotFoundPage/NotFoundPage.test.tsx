import React from "react";
import { render, screen } from "@testing-library/react";
import { BrowserRouter } from "react-router-dom";
import NotFoundPage from "./NotFoundPage";

const renderWithRouter = (component: React.ReactElement) => {
  return render(<BrowserRouter>{component}</BrowserRouter>);
};

describe("NotFoundPage", () => {
  test("renders 404 heading", () => {
    renderWithRouter(<NotFoundPage />);
    expect(screen.getByText("404 - Page Not Found")).toBeInTheDocument();
  });

  test("renders description message", () => {
    renderWithRouter(<NotFoundPage />);
    expect(
      screen.getByText(/The page you're looking for doesn't exist/)
    ).toBeInTheDocument();
  });

  test("renders back to overview link", () => {
    renderWithRouter(<NotFoundPage />);
    const link = screen.getByText("Back to Overview");
    expect(link).toBeInTheDocument();
    expect(link.closest("a")).toHaveAttribute("href", "/");
  });

  test("renders alert icon", () => {
    const { container } = renderWithRouter(<NotFoundPage />);
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });
});
