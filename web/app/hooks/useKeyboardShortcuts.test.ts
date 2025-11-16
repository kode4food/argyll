import { renderHook } from "@testing-library/react";
import { useKeyboardShortcuts } from "./useKeyboardShortcuts";

describe("useKeyboardShortcuts", () => {
  let handler1: jest.Mock;
  let handler2: jest.Mock;

  beforeEach(() => {
    handler1 = jest.fn();
    handler2 = jest.fn();
  });

  test("triggers handler on matching key", () => {
    renderHook(() =>
      useKeyboardShortcuts([
        { key: "a", description: "Test", handler: handler1 },
      ])
    );

    const event = new KeyboardEvent("keydown", { key: "a" });
    document.dispatchEvent(event);

    expect(handler1).toHaveBeenCalledTimes(1);
  });

  test("does not trigger on non-matching key", () => {
    renderHook(() =>
      useKeyboardShortcuts([
        { key: "a", description: "Test", handler: handler1 },
      ])
    );

    const event = new KeyboardEvent("keydown", { key: "b" });
    document.dispatchEvent(event);

    expect(handler1).not.toHaveBeenCalled();
  });

  test("respects ctrl modifier", () => {
    renderHook(() =>
      useKeyboardShortcuts([
        { key: "s", ctrl: true, description: "Save", handler: handler1 },
      ])
    );

    const event1 = new KeyboardEvent("keydown", { key: "s" });
    document.dispatchEvent(event1);
    expect(handler1).not.toHaveBeenCalled();

    const event2 = new KeyboardEvent("keydown", { key: "s", ctrlKey: true });
    document.dispatchEvent(event2);
    expect(handler1).toHaveBeenCalledTimes(1);
  });

  test("respects meta modifier", () => {
    renderHook(() =>
      useKeyboardShortcuts([
        { key: "k", meta: true, description: "Command", handler: handler1 },
      ])
    );

    const event1 = new KeyboardEvent("keydown", { key: "k" });
    document.dispatchEvent(event1);
    expect(handler1).not.toHaveBeenCalled();

    const event2 = new KeyboardEvent("keydown", { key: "k", metaKey: true });
    document.dispatchEvent(event2);
    expect(handler1).toHaveBeenCalledTimes(1);
  });

  test("respects shift modifier", () => {
    renderHook(() =>
      useKeyboardShortcuts([
        { key: "A", shift: true, description: "Shift A", handler: handler1 },
      ])
    );

    const event1 = new KeyboardEvent("keydown", { key: "A" });
    document.dispatchEvent(event1);
    expect(handler1).not.toHaveBeenCalled();

    const event2 = new KeyboardEvent("keydown", { key: "A", shiftKey: true });
    document.dispatchEvent(event2);
    expect(handler1).toHaveBeenCalledTimes(1);
  });

  test("handles multiple shortcuts", () => {
    renderHook(() =>
      useKeyboardShortcuts([
        { key: "a", description: "Action A", handler: handler1 },
        { key: "b", description: "Action B", handler: handler2 },
      ])
    );

    const event1 = new KeyboardEvent("keydown", { key: "a" });
    document.dispatchEvent(event1);
    expect(handler1).toHaveBeenCalledTimes(1);
    expect(handler2).not.toHaveBeenCalled();

    const event2 = new KeyboardEvent("keydown", { key: "b" });
    document.dispatchEvent(event2);
    expect(handler1).toHaveBeenCalledTimes(1);
    expect(handler2).toHaveBeenCalledTimes(1);
  });

  test("ignores shortcuts when disabled", () => {
    renderHook(() =>
      useKeyboardShortcuts(
        [{ key: "a", description: "Test", handler: handler1 }],
        false
      )
    );

    const event = new KeyboardEvent("keydown", { key: "a" });
    document.dispatchEvent(event);

    expect(handler1).not.toHaveBeenCalled();
  });

  test("ignores shortcuts when input is focused", () => {
    const input = document.createElement("input");
    document.body.appendChild(input);
    input.focus();

    renderHook(() =>
      useKeyboardShortcuts([
        { key: "a", description: "Test", handler: handler1 },
      ])
    );

    const event = new KeyboardEvent("keydown", { key: "a" });
    document.dispatchEvent(event);

    expect(handler1).not.toHaveBeenCalled();

    document.body.removeChild(input);
  });

  test("ignores shortcuts when textarea is focused", () => {
    const textarea = document.createElement("textarea");
    document.body.appendChild(textarea);
    textarea.focus();

    renderHook(() =>
      useKeyboardShortcuts([
        { key: "a", description: "Test", handler: handler1 },
      ])
    );

    const event = new KeyboardEvent("keydown", { key: "a" });
    document.dispatchEvent(event);

    expect(handler1).not.toHaveBeenCalled();

    document.body.removeChild(textarea);
  });

  test("ignores shortcuts when contenteditable is focused", () => {
    const div = document.createElement("div");
    div.setAttribute("contenteditable", "true");
    document.body.appendChild(div);
    div.focus();

    renderHook(() =>
      useKeyboardShortcuts([
        { key: "a", description: "Test", handler: handler1 },
      ])
    );

    const event = new KeyboardEvent("keydown", { key: "a" });
    document.dispatchEvent(event);

    expect(handler1).not.toHaveBeenCalled();

    document.body.removeChild(div);
  });

  test("triggers / shortcut even when input not focused", () => {
    renderHook(() =>
      useKeyboardShortcuts([
        { key: "/", description: "Search", handler: handler1 },
      ])
    );

    const event = new KeyboardEvent("keydown", { key: "/" });
    document.dispatchEvent(event);

    expect(handler1).toHaveBeenCalledTimes(1);
  });

  test("does not trigger / shortcut when input is focused", () => {
    const input = document.createElement("input");
    document.body.appendChild(input);
    input.focus();

    renderHook(() =>
      useKeyboardShortcuts([
        { key: "/", description: "Search", handler: handler1 },
      ])
    );

    const event = new KeyboardEvent("keydown", { key: "/" });
    document.dispatchEvent(event);

    expect(handler1).not.toHaveBeenCalled();

    document.body.removeChild(input);
  });

  test("triggers ? shortcut even when input not focused", () => {
    renderHook(() =>
      useKeyboardShortcuts([
        { key: "?", description: "Help", handler: handler1 },
      ])
    );

    const event = new KeyboardEvent("keydown", { key: "?" });
    document.dispatchEvent(event);

    expect(handler1).toHaveBeenCalledTimes(1);
  });

  test("triggers Escape even when input is focused", () => {
    const input = document.createElement("input");
    document.body.appendChild(input);
    input.focus();

    renderHook(() =>
      useKeyboardShortcuts([
        { key: "Escape", description: "Close", handler: handler1 },
      ])
    );

    const event = new KeyboardEvent("keydown", { key: "Escape" });
    document.dispatchEvent(event);

    expect(handler1).toHaveBeenCalledTimes(1);

    document.body.removeChild(input);
  });

  test("cleans up event listener on unmount", () => {
    const removeEventListenerSpy = jest.spyOn(document, "removeEventListener");

    const { unmount } = renderHook(() =>
      useKeyboardShortcuts([
        { key: "a", description: "Test", handler: handler1 },
      ])
    );

    unmount();

    expect(removeEventListenerSpy).toHaveBeenCalledWith(
      "keydown",
      expect.any(Function)
    );

    removeEventListenerSpy.mockRestore();
  });

  test("cleans up event listener when disabled", () => {
    const { rerender } = renderHook(
      ({ enabled }) =>
        useKeyboardShortcuts(
          [{ key: "a", description: "Test", handler: handler1 }],
          enabled
        ),
      { initialProps: { enabled: true } }
    );

    const event = new KeyboardEvent("keydown", { key: "a" });
    document.dispatchEvent(event);
    expect(handler1).toHaveBeenCalledTimes(1);

    rerender({ enabled: false });

    document.dispatchEvent(event);
    expect(handler1).toHaveBeenCalledTimes(1);
  });
});
