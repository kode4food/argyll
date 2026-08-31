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

jest.mock("@/app/components/molecules/ScriptEditor", () => ({
  __esModule: true,
  default: ({ value, onChange, language, readOnly }: any) => (
    <textarea
      data-testid="selector-editor"
      data-language={language}
      value={value}
      readOnly={readOnly}
      onChange={(e) => onChange(e.target.value)}
    />
  ),
}));

jest.mock("@/app/api", () => ({
  ...jest.requireActual("@/app/api"),
  api: {
    registerSpace: jest.fn(),
    previewSpace: jest.fn(),
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
        selector: {
          language: "lua",
          script: 'return value["domain"] == "risk"',
        },
        qbe: { domain: ["risk"] },
      },
      {
        id: "gold",
        name: "Gold",
        selector: {
          language: "lua",
          script: 'return value["tier"] == "gold"',
        },
      },
    ];
    stepsInStore = [
      {
        id: "score-customer",
        name: "Score Customer",
        type: "service",
        attributes: {},
        labels: { domain: "risk", tier: "gold" },
      },
      {
        id: "place-order",
        name: "Place Order",
        type: "service",
        attributes: {},
        labels: { domain: "trading" },
      },
    ];
    mockApi.registerSpace.mockImplementation(async (space) => space);
    mockApi.previewSpace.mockResolvedValue(["score-customer"]);
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

  const editScript = () => {
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.editScript") })
    );
  };

  const setScript = (script: string) => {
    fireEvent.change(screen.getByTestId("selector-editor"), {
      target: { value: script },
    });
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
    next();

    expect(screen.getByTestId("selector-editor")).toHaveValue(
      'return value["domain"] == "trading"'
    );
    expect(screen.getByTestId("selector-editor")).toHaveAttribute("readonly");
    await waitFor(() =>
      expect(mockApi.previewSpace).toHaveBeenCalledWith({
        id: "domain-trading",
        name: "trading domain",
        selector: {
          language: "lua",
          script: 'return value["domain"] == "trading"',
        },
        qbe: { domain: ["trading"] },
      })
    );
    expect(screen.getByText("1 Step in this Space")).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.save") })
    );

    await waitFor(() => {
      expect(mockApi.registerSpace).toHaveBeenCalledWith({
        id: "domain-trading",
        name: "trading domain",
        selector: {
          language: "lua",
          script: 'return value["domain"] == "trading"',
        },
        qbe: { domain: ["trading"] },
      });
      expect(setSpaceId).toHaveBeenCalledWith("domain-trading");
    });
  });

  test("counts matching Steps while the selector is drafted", async () => {
    open();
    startNew();
    addSelector("domain", "trading");

    await waitFor(() =>
      expect(mockApi.previewSpace).toHaveBeenLastCalledWith({
        id: "",
        name: "",
        selector: {
          language: "lua",
          script: 'return value["domain"] == "trading"',
        },
        qbe: { domain: ["trading"] },
      })
    );
    expect(screen.getByText("1 Step in this Space")).toBeInTheDocument();
  });

  test("escapes control characters in generated Lua", async () => {
    open();
    startNew();
    addSelector("domain", "bell");

    await waitFor(() =>
      expect(mockApi.previewSpace).toHaveBeenLastCalledWith(
        expect.objectContaining({
          selector: {
            language: "lua",
            script: 'return value["domain"] == "bell\\007"',
          },
        })
      )
    );
  });

  test("surfaces a failed preview", async () => {
    mockApi.previewSpace.mockRejectedValue(new Error("engine unreachable"));
    open();
    startNew();
    addSelector("domain", "trading");

    await waitFor(() =>
      expect(screen.getByText("engine unreachable")).toBeInTheDocument()
    );
  });

  test("removes a selector row", async () => {
    open();
    startNew();
    addSelector("domain", "trading");
    addSelector("tier", "gold");
    fireEvent.click(
      screen.getByRole("button", {
        name: `${t("spaceManager.removeSelector")} tier`,
      })
    );

    await waitFor(() =>
      expect(mockApi.previewSpace).toHaveBeenLastCalledWith(
        expect.objectContaining({ qbe: { domain: ["trading"] } })
      )
    );
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
    next();
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.save") })
    );

    await waitFor(() => {
      expect(mockApi.updateSpace).toHaveBeenCalledWith("risk", {
        id: "risk",
        name: "Risk Domain",
        description: "Risk steps",
        selector: {
          language: "lua",
          script: 'return value["domain"] == "risk"',
        },
        qbe: { domain: ["risk"] },
      });
      expect(setSpaceId).toHaveBeenCalledWith("risk");
    });
  });

  test("combines repeated selector keys as alternatives", async () => {
    open();
    fireEvent.click(screen.getByText("Risk"));
    addSelector("domain", "trading");
    addSelector("domain", "risk");
    next();
    next();
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.save") })
    );

    await waitFor(() => {
      expect(mockApi.updateSpace).toHaveBeenCalledWith("risk", {
        id: "risk",
        name: "Risk",
        description: "Risk steps",
        selector: {
          language: "lua",
          script:
            'return (value["domain"] == "risk" or value["domain"] == "trading")',
        },
        qbe: { domain: ["risk", "trading"] },
      });
    });
  });

  test("ejects QBE into an editable Lua selector", async () => {
    open();
    fireEvent.click(screen.getByText("Risk"));
    addSelector("domain", "trading");
    next();
    next();

    expect(screen.getByTestId("selector-editor")).toHaveAttribute("readonly");
    editScript();

    expect(screen.getByTestId("selector-editor")).toHaveValue(
      'return (value["domain"] == "risk" or value["domain"] == "trading")'
    );
    expect(screen.getByTestId("selector-editor")).not.toHaveAttribute(
      "readonly"
    );
    setScript('return value["tier"] == "gold"');
    await waitFor(() =>
      expect(mockApi.previewSpace).toHaveBeenLastCalledWith({
        id: "risk",
        name: "Risk",
        description: "Risk steps",
        selector: {
          language: "lua",
          script: 'return value["tier"] == "gold"',
        },
      })
    );
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.save") })
    );

    await waitFor(() => {
      expect(mockApi.updateSpace).toHaveBeenCalledWith("risk", {
        id: "risk",
        name: "Risk",
        description: "Risk steps",
        selector: {
          language: "lua",
          script: 'return value["tier"] == "gold"',
        },
      });
    });
  });

  test("previews the script once typing settles", async () => {
    open();
    fireEvent.click(screen.getByText("Risk"));
    editScript();
    await waitFor(() => expect(mockApi.previewSpace).toHaveBeenCalledTimes(1));

    setScript("return ");
    setScript('return value["tier"]');
    setScript('return value["tier"] == "gold"');

    await waitFor(() =>
      expect(mockApi.previewSpace).toHaveBeenLastCalledWith(
        expect.objectContaining({
          selector: {
            language: "lua",
            script: 'return value["tier"] == "gold"',
          },
        })
      )
    );
    expect(mockApi.previewSpace).toHaveBeenCalledTimes(2);
  });

  test("leaves the edited script as typed", async () => {
    open();
    fireEvent.click(screen.getByText("Risk"));
    editScript();

    const typed = '  return value["tier"] == "gold"\n\n';
    setScript(typed);
    await waitFor(() =>
      expect(mockApi.previewSpace).toHaveBeenLastCalledWith(
        expect.objectContaining({
          selector: { language: "lua", script: typed.trim() },
        })
      )
    );
    expect(screen.getByTestId("selector-editor")).toHaveValue(typed);
  });

  test("reports a selector that matches nothing", async () => {
    mockApi.previewSpace.mockResolvedValue([]);
    open();
    fireEvent.click(screen.getByText("Gold"));

    await waitFor(() =>
      expect(screen.getByText("No Steps in this Space")).toBeInTheDocument()
    );
  });

  test("opens an ejected Space on its editable script", async () => {
    open();
    fireEvent.click(screen.getByText("Gold"));

    const editor = screen.getByTestId("selector-editor");
    expect(editor).toHaveValue('return value["tier"] == "gold"');
    expect(editor).not.toHaveAttribute("readonly");
    await waitFor(() =>
      expect(screen.getByText("1 Step in this Space")).toBeInTheDocument()
    );
  });

  test("bypasses QBE with a JPath selector", async () => {
    open();
    startNew();
    editScript();
    fireEvent.click(
      screen.getByRole("button", { name: t("script.language.jpath") })
    );
    setScript('$.domain == "risk"');
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.back") })
    );
    fireEvent.change(
      screen.getByPlaceholderText(t("spaceManager.idPlaceholder")),
      { target: { value: "risk-script" } }
    );
    fireEvent.change(
      screen.getByPlaceholderText(t("spaceManager.namePlaceholder")),
      { target: { value: "Risk Script" } }
    );
    next();
    await waitFor(() =>
      expect(mockApi.previewSpace).toHaveBeenLastCalledWith({
        id: "risk-script",
        name: "Risk Script",
        selector: { language: "jpath", script: '$.domain == "risk"' },
      })
    );
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.save") })
    );

    await waitFor(() => {
      expect(mockApi.registerSpace).toHaveBeenCalledWith({
        id: "risk-script",
        name: "Risk Script",
        selector: { language: "jpath", script: '$.domain == "risk"' },
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
      screen.getByRole("button", { name: t("spaceManager.next") })
    ).toBeDisabled();
  });

  test("keeps save disabled for an unmodified Space", async () => {
    open();
    fireEvent.click(screen.getByText("Risk"));
    next();
    next();

    expect(
      screen.getByRole("button", { name: t("spaceManager.save") })
    ).toBeDisabled();
    await waitFor(() =>
      expect(screen.getByText("1 Step in this Space")).toBeInTheDocument()
    );
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

  test("excludes values already used for the same selector key", () => {
    open();
    startNew();
    addSelector("domain", "risk");
    addSelector("domain");
    fireEvent.click(screen.getAllByLabelText("Show suggestions")[3]);

    expect(screen.getByRole("option", { name: "trading" })).toBeInTheDocument();
    expect(
      screen.queryByRole("option", { name: "risk" })
    ).not.toBeInTheDocument();
  });

  test("allows keys already used by another selector row", () => {
    open();
    startNew();
    addSelector("domain", "risk");
    addSelector("");
    fireEvent.click(screen.getAllByLabelText("Show suggestions")[2]);

    expect(screen.getByRole("option", { name: "tier" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "domain" })).toBeInTheDocument();
  });
});
