import { render, screen, fireEvent } from "@testing-library/react";
import KeyboardShortcutsModal from "./KeyboardShortcutsModal";
import { t } from "@/app/testUtils/i18n";

jest.mock("@/app/hooks/useEscapeKey", () => ({
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

    expect(
      screen.queryByText(t("keyboardShortcuts.title"))
    ).not.toBeInTheDocument();
  });

  test("renders when open", () => {
    render(<KeyboardShortcutsModal isOpen={true} onClose={jest.fn()} />);

    expect(screen.getByText(t("keyboardShortcuts.title"))).toBeInTheDocument();
  });

  test("renders General section shortcuts", () => {
    render(<KeyboardShortcutsModal isOpen={true} onClose={jest.fn()} />);

    expect(
      screen.getByText(t("keyboardShortcuts.showShortcuts"))
    ).toBeInTheDocument();
    expect(
      screen.getByText(t("keyboardShortcuts.focusSearch"))
    ).toBeInTheDocument();
    expect(
      screen.getByText(t("keyboardShortcuts.closeModals"))
    ).toBeInTheDocument();
  });

  test("renders Navigation section shortcuts", () => {
    render(<KeyboardShortcutsModal isOpen={true} onClose={jest.fn()} />);

    expect(
      screen.getByText(t("keyboardShortcuts.navigateWithinLevel"))
    ).toBeInTheDocument();
    expect(
      screen.getByText(t("keyboardShortcuts.navigateBetweenLevels"))
    ).toBeInTheDocument();
    expect(
      screen.getByText(t("keyboardShortcuts.openStepEditor"))
    ).toBeInTheDocument();
  });

  test("renders section titles", () => {
    render(<KeyboardShortcutsModal isOpen={true} onClose={jest.fn()} />);

    expect(
      screen.getByText(t("keyboardShortcuts.sectionGeneral"))
    ).toBeInTheDocument();
    expect(
      screen.getByText(t("keyboardShortcuts.sectionNavigation"))
    ).toBeInTheDocument();
  });

  test("calls onClose when close button clicked", () => {
    const onClose = jest.fn();

    render(<KeyboardShortcutsModal isOpen={true} onClose={onClose} />);

    const closeButton = screen.getByRole("button", {
      name: t("keyboardShortcuts.close"),
    });
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
