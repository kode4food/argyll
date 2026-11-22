import { render } from "@testing-library/react";
import CollectIcon from "./CollectIcon";

describe("CollectIcon", () => {
  test("masks with the collect icon", () => {
    const { container } = render(<CollectIcon collect="all" title="All" />);
    const icon = container.firstChild as HTMLElement;
    expect(icon.style.maskImage).toBe("url(/icons/collect-all.svg)");
    expect(icon).toHaveAttribute("title", "All");
  });
});
