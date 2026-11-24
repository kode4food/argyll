import { render, screen } from "@testing-library/react";
import ChatTranscript from "./ChatTranscript";

describe("ChatTranscript", () => {
  test("shows the empty text", () => {
    render(
      <ChatTranscript
        messages={[]}
        isPending={false}
        error={null}
        emptyText="Ask away"
      />
    );
    expect(screen.getByText("Ask away")).toBeInTheDocument();
  });

  test("renders messages and errors", () => {
    render(
      <ChatTranscript
        messages={[
          { role: "user", content: "hi **there**" },
          { role: "assistant", content: "**hello**" },
        ]}
        isPending={false}
        error="no key"
        emptyText="Ask away"
      />
    );
    expect(screen.getByText("hi **there**")).toBeInTheDocument();
    expect(screen.getByText("hello").tagName).toBe("STRONG");
    expect(screen.getByRole("alert")).toHaveTextContent("no key");
    expect(screen.queryByText("Ask away")).not.toBeInTheDocument();
  });

  test("hides the empty text while pending", () => {
    render(
      <ChatTranscript messages={[]} isPending error={null} emptyText="Ask" />
    );
    expect(screen.queryByText("Ask")).not.toBeInTheDocument();
  });
});
