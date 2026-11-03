import { fireEvent, render, screen } from "@testing-library/react";
import { t } from "@/app/testUtils/i18n";
import CatalogMenu from "./CatalogMenu";

interface MockManagerProps {
  isOpen: boolean;
  onClose: () => void;
}

// A declaration, so jest.mock's hoisted factories can reach it
function mockManager(name: string) {
  const MockManager = ({ isOpen, onClose }: MockManagerProps) =>
    isOpen ? <button onClick={onClose}>{name} open</button> : null;
  return MockManager;
}

jest.mock("@/app/components/organisms/SpaceManager", () => ({
  __esModule: true,
  default: mockManager("Spaces"),
}));

jest.mock("@/app/components/organisms/FeedManager", () => ({
  __esModule: true,
  default: mockManager("Feeds"),
}));

const openMenu = () =>
  fireEvent.click(
    screen.getByRole("button", { name: t("flowCreate.catalogMenu") })
  );

describe("CatalogMenu", () => {
  test("lists the Catalog managers", () => {
    render(<CatalogMenu />);
    expect(screen.queryByRole("menu")).toBeNull();

    openMenu();
    expect(
      screen.getByRole("menuitem", { name: t("spaceManager.title") })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("menuitem", { name: t("feedManager.title") })
    ).toBeInTheDocument();
  });

  test.each([
    ["spaceManager.title", "Spaces open"],
    ["feedManager.title", "Feeds open"],
  ])("opens %s and closes it", (labelKey, opened) => {
    render(<CatalogMenu />);
    openMenu();
    fireEvent.click(screen.getByRole("menuitem", { name: t(labelKey) }));

    expect(screen.queryByRole("menu")).toBeNull();
    fireEvent.click(screen.getByText(opened));
    expect(screen.queryByText(opened)).toBeNull();
  });

  test("opens a manager from the keyboard", () => {
    render(<CatalogMenu />);
    const button = screen.getByRole("button", {
      name: t("flowCreate.catalogMenu"),
    });
    fireEvent.keyDown(button, { key: "ArrowDown" });
    fireEvent.keyDown(button, { key: "ArrowDown" });
    fireEvent.keyDown(button, { key: "Enter" });

    expect(screen.getByText("Feeds open")).toBeInTheDocument();
  });
});
