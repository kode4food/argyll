import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { Space, Step } from "@/app/api";
import { t } from "@/app/testUtils/i18n";
import SpaceManager from "./SpaceManager";

let spacesInStore: Space[] = [];
let stepsInStore: Step[] = [];
const loadSteps = jest.fn();

jest.mock("@/app/store/flowStore", () => ({
  useSpaces: () => spacesInStore,
  useSteps: () => stepsInStore,
  useSpaceSelection: () => ({ risk: new Set(["score-customer"]) }),
  useLoadSteps: () => loadSteps,
}));

jest.mock("@/app/api", () => ({
  ...jest.requireActual("@/app/api"),
  api: {
    registerSpace: jest.fn(),
    updateSpace: jest.fn(),
    unregisterSpace: jest.fn(),
  },
}));

import { api } from "@/app/api";

const mockApi = api as jest.Mocked<typeof api>;

describe("SpaceManager", () => {
  beforeEach(() => {
    jest.resetAllMocks();
    spacesInStore = [
      {
        id: "risk",
        name: "Risk",
        description: "Risk steps",
        selector: { match_labels: { domain: "risk" } },
      },
    ];
    stepsInStore = [
      {
        id: "score-customer",
        name: "Score Customer",
        type: "sync",
        attributes: {},
        labels: { domain: "risk", tier: "gold" },
      },
      {
        id: "place-order",
        name: "Place Order",
        type: "sync",
        attributes: {},
        labels: { domain: "trading" },
      },
    ];
  });

  const open = () => render(<SpaceManager isOpen onClose={jest.fn()} />);

  test("renders nothing when closed", () => {
    render(<SpaceManager isOpen={false} onClose={jest.fn()} />);

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  test("lists existing Spaces", () => {
    open();

    expect(screen.getByText("Risk")).toBeInTheDocument();
    expect(screen.getByText("risk")).toBeInTheDocument();
  });

  test("registers a new Space", async () => {
    open();

    fireEvent.change(
      screen.getByPlaceholderText(t("spaceManager.idPlaceholder")),
      { target: { value: "trading" } }
    );
    fireEvent.change(
      screen.getByPlaceholderText(t("spaceManager.namePlaceholder")),
      { target: { value: "Trading" } }
    );
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.addSelector") })
    );
    fireEvent.change(
      screen.getByPlaceholderText(t("stepEditor.labelKeyPlaceholder")),
      { target: { value: "domain" } }
    );
    fireEvent.change(
      screen.getByPlaceholderText(t("stepEditor.labelValuePlaceholder")),
      { target: { value: "trading" } }
    );
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.save") })
    );

    await waitFor(() => {
      expect(mockApi.registerSpace).toHaveBeenCalledWith({
        id: "trading",
        name: "Trading",
        selector: { match_labels: { domain: "trading" } },
      });
    });
    expect(loadSteps).not.toHaveBeenCalled();
  });

  test("loads a Space for editing and updates it", async () => {
    open();
    fireEvent.click(screen.getByText("Risk"));

    expect(
      screen.getByPlaceholderText(t("spaceManager.idPlaceholder"))
    ).toBeDisabled();

    fireEvent.change(
      screen.getByPlaceholderText(t("spaceManager.namePlaceholder")),
      { target: { value: "Risk Domain" } }
    );
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.save") })
    );

    await waitFor(() => {
      expect(mockApi.updateSpace).toHaveBeenCalledWith("risk", {
        id: "risk",
        name: "Risk Domain",
        description: "Risk steps",
        selector: { match_labels: { domain: "risk" } },
      });
    });
  });

  test("surfaces a conflict when deleting a Space in use", async () => {
    mockApi.unregisterSpace.mockRejectedValue(
      new Error("space in use: charge-card")
    );
    open();
    fireEvent.click(screen.getByText("Risk"));
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.delete") })
    );

    await waitFor(() => {
      expect(screen.getByText("space in use: charge-card")).toBeInTheDocument();
    });
  });

  test("requires an id and a name before saving", () => {
    open();

    expect(
      screen.getByRole("button", { name: t("spaceManager.save") })
    ).toBeDisabled();
  });

  test("requires at least one selector before saving", () => {
    open();
    fireEvent.change(
      screen.getByPlaceholderText(t("spaceManager.idPlaceholder")),
      { target: { value: "trading" } }
    );
    fireEvent.change(
      screen.getByPlaceholderText(t("spaceManager.namePlaceholder")),
      { target: { value: "Trading" } }
    );

    expect(
      screen.getByRole("button", { name: t("spaceManager.save") })
    ).toBeDisabled();
  });

  test("keeps the Space selected after saving it", async () => {
    open();
    fireEvent.click(screen.getByText("Risk"));
    fireEvent.change(
      screen.getByPlaceholderText(t("spaceManager.namePlaceholder")),
      { target: { value: "Risk Domain" } }
    );
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.save") })
    );

    await waitFor(() => expect(mockApi.updateSpace).toHaveBeenCalled());
    expect(
      screen.getByPlaceholderText(t("spaceManager.namePlaceholder"))
    ).toHaveValue("Risk Domain");
    expect(
      screen.getByRole("button", { name: t("spaceManager.delete") })
    ).toBeInTheDocument();
  });

  test("clears the form after deleting a Space", async () => {
    open();
    fireEvent.click(screen.getByText("Risk"));
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.delete") })
    );

    await waitFor(() =>
      expect(
        screen.getByPlaceholderText(t("spaceManager.namePlaceholder"))
      ).toHaveValue("")
    );
    expect(mockApi.unregisterSpace).toHaveBeenCalledWith("risk");
  });

  test("keeps save disabled for an unmodified Space", () => {
    open();
    fireEvent.click(screen.getByText("Risk"));

    expect(
      screen.getByRole("button", { name: t("spaceManager.save") })
    ).toBeDisabled();
  });

  test("enables save once the loaded Space changes", () => {
    open();
    fireEvent.click(screen.getByText("Risk"));
    fireEvent.change(
      screen.getByPlaceholderText(t("spaceManager.namePlaceholder")),
      { target: { value: "Risk Domain" } }
    );

    expect(
      screen.getByRole("button", { name: t("spaceManager.save") })
    ).toBeEnabled();
  });

  test("keeps save disabled while a selector row is incomplete", () => {
    open();
    fireEvent.change(
      screen.getByPlaceholderText(t("spaceManager.idPlaceholder")),
      { target: { value: "trading" } }
    );
    fireEvent.change(
      screen.getByPlaceholderText(t("spaceManager.namePlaceholder")),
      { target: { value: "Trading" } }
    );
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.addSelector") })
    );
    fireEvent.change(
      screen.getByPlaceholderText(t("stepEditor.labelKeyPlaceholder")),
      { target: { value: "domain" } }
    );

    expect(
      screen.getByRole("button", { name: t("spaceManager.save") })
    ).toBeDisabled();
  });

  test("suggests label keys from the catalog", () => {
    open();
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.addSelector") })
    );
    fireEvent.click(screen.getAllByLabelText("Show suggestions")[0]);

    expect(screen.getByRole("option", { name: "domain" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "tier" })).toBeInTheDocument();
  });

  test("suggests values for the chosen key", () => {
    open();
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.addSelector") })
    );
    fireEvent.change(
      screen.getByPlaceholderText(t("stepEditor.labelKeyPlaceholder")),
      { target: { value: "domain" } }
    );
    fireEvent.click(screen.getAllByLabelText("Show suggestions")[1]);

    expect(screen.getByRole("option", { name: "risk" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "trading" })).toBeInTheDocument();
  });

  test("excludes keys already used by another row", () => {
    open();
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.addSelector") })
    );
    fireEvent.change(
      screen.getByPlaceholderText(t("stepEditor.labelKeyPlaceholder")),
      { target: { value: "domain" } }
    );
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.addSelector") })
    );
    fireEvent.click(screen.getAllByLabelText("Show suggestions")[2]);

    expect(screen.getByRole("option", { name: "tier" })).toBeInTheDocument();
    expect(
      screen.queryByRole("option", { name: "domain" })
    ).not.toBeInTheDocument();
  });
});
