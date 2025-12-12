import React from "react";
import { render } from "@testing-library/react";
import Spinner from "./Spinner";
import styles from "./Spinner.module.css";

describe("Spinner", () => {
  test("renders with default props", () => {
    const { container } = render(<Spinner />);
    const spinner = container.querySelector("div");
    expect(spinner).toBeInTheDocument();
    expect(spinner?.className).toContain(styles.spinner);
    expect(spinner?.className).toContain(styles.spinnerMd);
    expect(spinner?.className).toContain(styles.spinnerPrimary);
  });

  test("renders with small size", () => {
    const { container } = render(<Spinner size="sm" />);
    const spinner = container.querySelector("div");
    expect(spinner?.className).toContain(styles.spinnerSm);
  });

  test("renders with medium size", () => {
    const { container } = render(<Spinner size="md" />);
    const spinner = container.querySelector("div");
    expect(spinner?.className).toContain(styles.spinnerMd);
  });

  test("renders with large size", () => {
    const { container } = render(<Spinner size="lg" />);
    const spinner = container.querySelector("div");
    expect(spinner?.className).toContain(styles.spinnerLg);
  });

  test("renders with primary color", () => {
    const { container } = render(<Spinner color="primary" />);
    const spinner = container.querySelector("div");
    expect(spinner?.className).toContain(styles.spinnerPrimary);
  });

  test("renders with white color", () => {
    const { container } = render(<Spinner color="white" />);
    const spinner = container.querySelector("div");
    expect(spinner?.className).toContain(styles.spinnerWhite);
  });
});
