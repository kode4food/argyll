import { fireEvent, render, screen } from "@testing-library/react";
import ChatComposer, { type ChatComposerProps } from "./ChatComposer";

describe("ChatComposer", () => {
  const onChange = jest.fn();
  const onSubmit = jest.fn();

  beforeEach(() => jest.clearAllMocks());

  const renderComposer = (props: Partial<ChatComposerProps> = {}) =>
    render(
      <ChatComposer
        value="hi"
        isDisabled={false}
        placeholder="Message"
        sendLabel="Send"
        onChange={onChange}
        onSubmit={onSubmit}
        {...props}
      />
    );

  test("submits on Enter", () => {
    renderComposer();
    fireEvent.keyDown(screen.getByLabelText("Message"), { key: "Enter" });
    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  test("keeps Shift+Enter for new lines", () => {
    renderComposer();
    fireEvent.keyDown(screen.getByLabelText("Message"), {
      key: "Enter",
      shiftKey: true,
    });
    expect(onSubmit).not.toHaveBeenCalled();
  });

  test("submits from the send button", () => {
    renderComposer();
    fireEvent.click(screen.getByLabelText("Send"));
    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  test("reports edits", () => {
    renderComposer();
    fireEvent.change(screen.getByLabelText("Message"), {
      target: { value: "hello" },
    });
    expect(onChange).toHaveBeenCalledWith("hello");
  });

  test("blocks blank or disabled sends", () => {
    renderComposer({ value: "  " });
    fireEvent.keyDown(screen.getByLabelText("Message"), { key: "Enter" });
    expect(screen.getByLabelText("Send")).toBeDisabled();

    renderComposer({ isDisabled: true });
    expect(screen.getAllByLabelText("Message")[1]).toBeDisabled();
    expect(onSubmit).not.toHaveBeenCalled();
  });
});
