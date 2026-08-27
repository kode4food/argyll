import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { Space, Step } from "@/app/api";
import { t } from "@/app/testUtils/i18n";
import SpaceManager from "./SpaceManager";

let spacesInStore: Space[] = [];
let stepsInStore: Step[] = [];
const loadSteps = jest.fn();
const setSpaceId = jest.fn();

jest.mock("@/app/contexts/UIContext", () => ({
  useUI: () => ({ setSpaceId }),
}));

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
        selector: { domain: "risk" },
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
    mockApi.registerSpace.mockImplementation(async (space) => space);
    mockApi.updateSpace.mockImplementation(async (_, space) => space);
  });

  const open = (onClose = jest.fn()) =>
    render(<SpaceManager isOpen onClose={onClose} />);

  const startNew = () => {
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.new") })
    );
  };

  const addSelector = (key: string, value = "") => {
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.addSelector") })
    );
    const keys = screen.getAllByPlaceholderText(
      t("stepEditor.labelKeyPlaceholder")
    );
    const values = screen.getAllByPlaceholderText(
      t("stepEditor.labelValuePlaceholder")
    );
    fireEvent.change(keys.at(-1)!, { target: { value: key } });
    if (value) fireEvent.change(values.at(-1)!, { target: { value } });
  };

  const next = () => {
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.next") })
    );
  };

  test("renders nothing when closed", () => {
    render(<SpaceManager isOpen={false} onClose={jest.fn()} />);

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  test("lists existing Spaces", () => {
    open();

    expect(
      screen.getByRole("button", { name: "Risk risk" })
    ).toBeInTheDocument();
    expect(screen.getByText("risk")).toBeInTheDocument();
  });

  test("registers a new Space through selector and details", async () => {
    open();
    startNew();
    addSelector("domain", "trading");
    next();

    expect(
      screen.getByPlaceholderText(t("spaceManager.idPlaceholder"))
    ).toHaveValue("domain-trading");
    expect(
      screen.getByPlaceholderText(t("spaceManager.namePlaceholder"))
    ).toHaveValue("trading domain");
    expect(
      screen.getByPlaceholderText("Steps matching domain=trading")
    ).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.save") })
    );

    await waitFor(() => {
      expect(mockApi.registerSpace).toHaveBeenCalledWith({
        id: "domain-trading",
        name: "trading domain",
        selector: { domain: "trading" },
      });
      expect(setSpaceId).toHaveBeenCalledWith("domain-trading");
    });
  });

  test("generates details from the selector", () => {
    open();
    startNew();
    addSelector("domain", "credit-card");
    fireEvent.change(
      screen.getByPlaceholderText(t("stepEditor.labelValuePlaceholder")),
      { target: { value: "payments" } }
    );
    next();

    expect(
      screen.getByPlaceholderText(t("spaceManager.idPlaceholder"))
    ).toHaveValue("domain-payments");
    expect(
      screen.getByPlaceholderText(t("spaceManager.namePlaceholder"))
    ).toHaveValue("payments domain");
  });

  test("loads and updates an existing Space", async () => {
    open();
    fireEvent.click(screen.getByText("Risk"));
    next();

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
        selector: { domain: "risk" },
      });
      expect(setSpaceId).toHaveBeenCalledWith("risk");
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

  test("requires a complete selector before showing details", () => {
    open();
    startNew();

    expect(
      screen.getByRole("button", { name: t("spaceManager.next") })
    ).toBeDisabled();
    addSelector("domain");
    expect(
      screen.getByRole("button", { name: t("spaceManager.next") })
    ).toBeDisabled();
  });

  test("requires an id and name before saving", () => {
    open();
    startNew();
    addSelector("domain", "risk");
    next();
    fireEvent.change(
      screen.getByPlaceholderText(t("spaceManager.idPlaceholder")),
      { target: { value: "" } }
    );

    expect(
      screen.getByRole("button", { name: t("spaceManager.save") })
    ).toBeDisabled();
  });

  test("keeps save disabled for an unmodified Space", () => {
    open();
    fireEvent.click(screen.getByText("Risk"));
    next();

    expect(
      screen.getByRole("button", { name: t("spaceManager.save") })
    ).toBeDisabled();
  });

  test("cancel abandons an unfinished setup", async () => {
    const onClose = jest.fn();
    const view = open(onClose);
    startNew();
    addSelector("domain", "risk");
    next();
    fireEvent.change(
      screen.getByPlaceholderText("Steps matching domain=risk"),
      { target: { value: "Draft description" } }
    );
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.cancel") })
    );
    expect(onClose).toHaveBeenCalled();

    view.unmount();
    open();

    expect(
      screen.getByRole("button", { name: "Risk risk" })
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(
        screen.getByRole("button", { name: t("spaceManager.new") })
      ).toHaveFocus()
    );
    expect(
      screen.queryByRole("button", { name: t("spaceManager.back") })
    ).not.toBeInTheDocument();
  });

  test("suggests catalog label values", () => {
    open();
    startNew();
    addSelector("domain");
    fireEvent.click(screen.getAllByLabelText("Show suggestions")[1]);

    expect(screen.getByRole("option", { name: "risk" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "trading" })).toBeInTheDocument();
  });

  test("excludes keys already used by another selector row", () => {
    open();
    startNew();
    addSelector("domain", "risk");
    addSelector("");
    fireEvent.click(screen.getAllByLabelText("Show suggestions")[2]);

    expect(screen.getByRole("option", { name: "tier" })).toBeInTheDocument();
    expect(
      screen.queryByRole("option", { name: "domain" })
    ).not.toBeInTheDocument();
  });
});
