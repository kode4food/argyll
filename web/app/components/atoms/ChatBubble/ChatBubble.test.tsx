import { render, screen } from "@testing-library/react";
import ChatBubble from "./ChatBubble";
import styles from "./ChatBubble.module.css";

describe("ChatBubble", () => {
  test("renders content with its variant", () => {
    render(<ChatBubble variant="user">hello</ChatBubble>);
    const bubble = screen.getByText("hello");
    expect(bubble.className).toContain(styles.bubble);
    expect(bubble.className).toContain(styles.user);
  });
});
