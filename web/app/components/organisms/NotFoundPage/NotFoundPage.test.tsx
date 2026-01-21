import React from "react";
import { screen } from "@testing-library/react";
import NotFoundPage from "./NotFoundPage";
import { t } from "@/app/testUtils/i18n";
import { renderWithRouter } from "@/app/testUtils/render";

describe("NotFoundPage", () => {
  test("renders 404 heading", () => {
    renderWithRouter(<NotFoundPage />);
    expect(screen.getByText(t("notFound.title"))).toBeInTheDocument();
  });

  test("renders description message", () => {
    renderWithRouter(<NotFoundPage />);
    expect(screen.getByText(t("notFound.description"))).toBeInTheDocument();
  });

  test("renders back to overview link", () => {
    renderWithRouter(<NotFoundPage />);
    const link = screen.getByText(t("common.backToOverview"));
    expect(link).toBeInTheDocument();
    expect(link.closest("a")).toHaveAttribute("href", "/");
  });

  test("renders alert icon", () => {
    const { container } = renderWithRouter(<NotFoundPage />);
    const svg = container.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });
});
