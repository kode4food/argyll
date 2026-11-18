import {
  render,
  screen,
  fireEvent,
  waitFor,
  act,
} from "@testing-library/react";
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

  test.each([
    { height: 100, side: "below" },
    { height: 180, side: "above" },
  ])("places a $height px tooltip $side", ({ height, side }) => {
    render(
      <Tooltip trigger={<button>Hover me</button>}>
        <div>Usage table</div>
      </Tooltip>
    );
    const trigger = screen.getByRole("button").parentElement!;
    const tooltip = screen.getByText("Usage table").parentElement!;
    jest
      .spyOn(trigger, "getBoundingClientRect")
      .mockReturnValue(new DOMRect(0, window.innerHeight - 170, 20, 30));
    jest
      .spyOn(tooltip, "getBoundingClientRect")
      .mockReturnValue(new DOMRect(0, 0, 100, height));

    fireEvent.mouseEnter(trigger);

    expect(tooltip).toHaveClass(side);
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

  test("hides tooltip when hideTooltips event is dispatched", async () => {
    render(
      <Tooltip trigger={<button>Hover me</button>}>
        <div>Tooltip content</div>
      </Tooltip>
    );

    const trigger = screen.getByText("Hover me").parentElement;
    fireEvent.mouseEnter(trigger!);

    const tooltip = document.querySelector(".portal");
    expect(tooltip?.className).toContain("visible");

    // Dispatch hideTooltips event
    const event = new Event("hideTooltips");
    await act(async () => {
      document.dispatchEvent(event);
    });

    await waitFor(() => {
      expect(tooltip?.className).not.toContain("visible");
    });
  });

  test("shows inside an enclosing dialog, above the page", () => {
    render(
      <dialog open>
        <Tooltip trigger={<button>Hover me</button>}>
          <div>Tooltip content</div>
        </Tooltip>
      </dialog>
    );

    fireEvent.mouseEnter(screen.getByText("Hover me").parentElement!);
    expect(
      screen.getByText("Tooltip content").closest("dialog")
    ).not.toBeNull();
  });
});
