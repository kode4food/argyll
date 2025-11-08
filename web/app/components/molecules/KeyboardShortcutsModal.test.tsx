import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";
import KeyboardShortcutsModal from "./KeyboardShortcutsModal";

jest.mock("../../hooks/useEscapeKey", () => ({
  useEscapeKey: jest.fn((isActive, callback) => {
    if (isActive) {
      const handler = (e: KeyboardEvent) => {
        if (e.key === "Escape") callback();
      };
      document.addEventListener("keydown", handler);
      return () => document.removeEventListener("keydown", handler);
    }
  }),
}));

describe("KeyboardShortcutsModal", () => {
  test("does not render when closed", () => {
    render(<KeyboardShortcutsModal isOpen={false} onClose={jest.fn()} />);

    expect(screen.queryByText("Keyboard Shortcuts")).not.toBeInTheDocument();
  });

  test("renders when open", () => {
    render(<KeyboardShortcutsModal isOpen={true} onClose={jest.fn()} />);

    expect(screen.getByText("Keyboard Shortcuts")).toBeInTheDocument();
  });

  test("renders General section shortcuts", () => {
    render(<KeyboardShortcutsModal isOpen={true} onClose={jest.fn()} />);

    expect(screen.getByText("Show keyboard shortcuts")).toBeInTheDocument();
    expect(screen.getByText("Focus search")).toBeInTheDocument();
    expect(
      screen.getByText("Close modals / Deselect step")
    ).toBeInTheDocument();
  });

  test("renders Navigation section shortcuts", () => {
    render(<KeyboardShortcutsModal isOpen={true} onClose={jest.fn()} />);

    expect(
      screen.getByText("Navigate within dependency level")
    ).toBeInTheDocument();
    expect(
      screen.getByText("Navigate between dependency levels")
    ).toBeInTheDocument();
    expect(
      screen.getByText("Open step editor (script steps)")
    ).toBeInTheDocument();
  });

  test("renders section titles", () => {
    render(<KeyboardShortcutsModal isOpen={true} onClose={jest.fn()} />);

    expect(screen.getByText("General")).toBeInTheDocument();
    expect(screen.getByText("Navigation")).toBeInTheDocument();
  });

  test("calls onClose when close button clicked", () => {
    const onClose = jest.fn();

    render(<KeyboardShortcutsModal isOpen={true} onClose={onClose} />);

    const closeButton = screen.getByRole("button", { name: "Close" });
    fireEvent.click(closeButton);

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  test("displays keyboard keys in kbd elements", () => {
    render(<KeyboardShortcutsModal isOpen={true} onClose={jest.fn()} />);

    const kbdElements = document.querySelectorAll("kbd");
    expect(kbdElements.length).toBeGreaterThan(0);
  });

  test("renders question mark shortcut", () => {
    render(<KeyboardShortcutsModal isOpen={true} onClose={jest.fn()} />);

    const shortcuts = screen.getAllByText("?");
    expect(shortcuts.length).toBeGreaterThan(0);
  });

  test("renders slash shortcut", () => {
    render(<KeyboardShortcutsModal isOpen={true} onClose={jest.fn()} />);

    const shortcuts = screen.getAllByText("/");
    expect(shortcuts.length).toBeGreaterThan(0);
  });

  test("renders escape shortcut", () => {
    render(<KeyboardShortcutsModal isOpen={true} onClose={jest.fn()} />);

    expect(screen.getByText("Esc")).toBeInTheDocument();
  });

  test("renders arrow shortcuts", () => {
    render(<KeyboardShortcutsModal isOpen={true} onClose={jest.fn()} />);

    expect(screen.getByText("↑")).toBeInTheDocument();
    expect(screen.getByText("↓")).toBeInTheDocument();
    expect(screen.getByText("←")).toBeInTheDocument();
    expect(screen.getByText("→")).toBeInTheDocument();
  });

  test("renders Enter shortcut", () => {
    render(<KeyboardShortcutsModal isOpen={true} onClose={jest.fn()} />);

    expect(screen.getByText("Enter")).toBeInTheDocument();
  });

  test("groups shortcuts by section", () => {
    const { container } = render(
      <KeyboardShortcutsModal isOpen={true} onClose={jest.fn()} />
    );

    const sections = container.querySelectorAll("[class*='section']");
    expect(sections.length).toBeGreaterThanOrEqual(2);
  });
});
