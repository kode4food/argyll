import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";
import Tooltip from "./Tooltip";

describe("Tooltip", () => {
  test("renders trigger element", () => {
    render(
      <Tooltip trigger={<button>Hover me</button>}>
        <div>Tooltip content</div>
      </Tooltip>
    );

    expect(screen.getByText("Hover me")).toBeInTheDocument();
  });

  test("does not show tooltip initially", () => {
    render(
      <Tooltip trigger={<button>Hover me</button>}>
        <div>Tooltip content</div>
      </Tooltip>
    );

    const tooltip = document.querySelector(".portal");
    expect(tooltip?.className).not.toContain("visible");
  });

  test("shows tooltip on mouse enter", () => {
    render(
      <Tooltip trigger={<button>Hover me</button>}>
        <div>Tooltip content</div>
      </Tooltip>
    );

    const trigger = screen.getByText("Hover me").parentElement;
    fireEvent.mouseEnter(trigger!);

    const tooltip = document.querySelector(".portal");
    expect(tooltip?.className).toContain("visible");
  });

  test("hides tooltip on mouse leave", () => {
    render(
      <Tooltip trigger={<button>Hover me</button>}>
        <div>Tooltip content</div>
      </Tooltip>
    );

    const trigger = screen.getByText("Hover me").parentElement;
    fireEvent.mouseEnter(trigger!);
    fireEvent.mouseLeave(trigger!);

    const tooltip = document.querySelector(".portal");
    expect(tooltip?.className).not.toContain("visible");
  });

  test("renders tooltip content", () => {
    render(
      <Tooltip trigger={<button>Hover me</button>}>
        <div>Tooltip content</div>
      </Tooltip>
    );

    expect(screen.getByText("Tooltip content")).toBeInTheDocument();
  });

  test("positions tooltip using fixed positioning", () => {
    render(
      <Tooltip trigger={<button>Hover me</button>}>
        <div>Tooltip content</div>
      </Tooltip>
    );

    const trigger = screen.getByText("Hover me").parentElement;
    fireEvent.mouseEnter(trigger!);

    const tooltip = document.querySelector(".portal") as HTMLElement;
    expect(tooltip?.style.position).toBe("fixed");
  });

  test("listens for hideTooltips custom event", () => {
    const addEventListenerSpy = jest.spyOn(document, "addEventListener");

    render(
      <Tooltip trigger={<button>Hover me</button>}>
        <div>Tooltip content</div>
      </Tooltip>
    );

    expect(addEventListenerSpy).toHaveBeenCalledWith(
      "hideTooltips",
      expect.any(Function)
    );

    addEventListenerSpy.mockRestore();
  });

  test("creates portal in document body", () => {
    render(
      <Tooltip trigger={<button>Hover me</button>}>
        <div>Tooltip content</div>
      </Tooltip>
    );

    const portalInBody = document.body.querySelector(".portal");
    expect(portalInBody).toBeInTheDocument();
  });

  test("cleans up event listener on unmount", () => {
    const { unmount } = render(
      <Tooltip trigger={<button>Hover me</button>}>
        <div>Tooltip content</div>
      </Tooltip>
    );

    const removeEventListenerSpy = jest.spyOn(document, "removeEventListener");
    unmount();

    expect(removeEventListenerSpy).toHaveBeenCalledWith(
      "hideTooltips",
      expect.any(Function)
    );

    removeEventListenerSpy.mockRestore();
  });
});
