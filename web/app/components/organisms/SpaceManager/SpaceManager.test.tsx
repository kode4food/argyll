import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { Space, Step } from "@/app/api";
import { t } from "@/app/testUtils/i18n";
import { addTag, suggestionsOf } from "@/app/testUtils/tags";
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
const GENERATED_SELECTOR = {
  language: "lua" as const,
  script: "return engine_generated",
};

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
        qbe: [["domain:risk"]],
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
        tags: ["domain:risk", "tier:gold"],
      },
      {
        id: "place-order",
        name: "Place Order",
        type: "service",
        attributes: {},
        tags: ["domain:trading", "domain:risk"],
      },
    ];
    mockApi.registerSpace.mockImplementation(async (space) => space);
    mockApi.previewSpace.mockImplementation(async (space) => ({
      space: { ...space, selector: space.selector ?? GENERATED_SELECTOR },
      step_ids: ["score-customer"],
    }));
    mockApi.updateSpace.mockImplementation(async (_, space) => space);
  });

  const open = (onClose = jest.fn()) =>
    render(<SpaceManager isOpen onClose={onClose} />);

  const startNew = () => {
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.new") })
    );
  };

  const addSelector = (tag: string) =>
    addTag(screen.getByLabelText(t("stepEditor.tagPlaceholder")), tag);

  const next = () => {
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.next") })
    );
  };

  const nextWhenEnabled = async () => {
    await waitFor(() =>
      expect(
        screen.getByRole("button", { name: t("spaceManager.next") })
      ).not.toBeDisabled()
    );
    next();
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

  test("focuses the selector when starting a new Space", () => {
    open();
    startNew();

    expect(document.querySelector(".tagify__input")).toHaveFocus();
  });

  test("prefills the id and name from the selector tags", async () => {
    open();
    startNew();
    await addSelector("domain:trading");
    await addSelector("tier_gold");
    next();

    expect(
      screen.getByPlaceholderText(t("spaceManager.idPlaceholder"))
    ).toHaveValue("domain-trading-tier-gold");
    expect(
      screen.getByPlaceholderText(t("spaceManager.namePlaceholder"))
    ).toHaveValue("Domain Trading Tier Gold");
  });

  test("registers a new Space through selector and details", async () => {
    open();
    startNew();
    await addSelector("domain:trading");
    next();

    expect(
      screen.getByPlaceholderText("Steps matching domain:trading")
    ).toBeInTheDocument();
    fireEvent.change(
      screen.getByPlaceholderText(t("spaceManager.idPlaceholder")),
      { target: { value: "domain-trading" } }
    );
    fireEvent.change(
      screen.getByPlaceholderText(t("spaceManager.namePlaceholder")),
      { target: { value: "trading" } }
    );
    await nextWhenEnabled();

    expect(screen.getByTestId("selector-editor")).toHaveValue(
      GENERATED_SELECTOR.script
    );
    expect(screen.getByTestId("selector-editor")).toHaveAttribute("readonly");
    await waitFor(() =>
      expect(mockApi.previewSpace).toHaveBeenCalledWith({
        id: "domain-trading",
        name: "trading",
        qbe: [["domain:trading"]],
      })
    );
    expect(screen.getByText("1 Step in this Space")).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.save") })
    );

    await waitFor(() => {
      expect(mockApi.registerSpace).toHaveBeenCalledWith({
        id: "domain-trading",
        name: "trading",
        qbe: [["domain:trading"]],
      });
      expect(setSpaceId).toHaveBeenCalledWith("domain-trading");
    });
  });

  test("counts matching Steps while the selector is drafted", async () => {
    open();
    startNew();
    await addSelector("domain:trading");

    await waitFor(() =>
      expect(mockApi.previewSpace).toHaveBeenLastCalledWith({
        id: "",
        name: "",
        qbe: [["domain:trading"]],
      })
    );
    expect(screen.getByText("1 Step in this Space")).toBeInTheDocument();
  });

  test("sends control characters as QBE without generating Lua", async () => {
    open();
    startNew();
    await addSelector("domain:bell");

    await waitFor(() =>
      expect(mockApi.previewSpace).toHaveBeenLastCalledWith(
        expect.objectContaining({
          qbe: [["domain:bell"]],
        })
      )
    );
  });

  test("surfaces a failed preview", async () => {
    mockApi.previewSpace.mockRejectedValue(new Error("engine unreachable"));
    open();
    startNew();
    await addSelector("domain:trading");

    await waitFor(() =>
      expect(screen.getByText("engine unreachable")).toBeInTheDocument()
    );
  });

  test("removes a selector tag", async () => {
    open();
    startNew();
    await addSelector("domain:trading");
    await addSelector("tier:gold");
    fireEvent.click(
      screen.getByRole("button", {
        name: `${t("spaceManager.removeSelector")} tier:gold`,
      })
    );

    await waitFor(() =>
      expect(mockApi.previewSpace).toHaveBeenLastCalledWith(
        expect.objectContaining({ qbe: [["domain:trading"]] })
      )
    );
  });

  const addAlternative = (tag: string) => {
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.addAlternative") })
    );
    const inputs = screen.getAllByLabelText(t("stepEditor.tagPlaceholder"));
    return addTag(inputs[inputs.length - 1], tag);
  };

  test("ORs an added alternative", async () => {
    open();
    startNew();
    await addSelector("domain:trading");
    await addAlternative("domain:risk");

    expect(screen.getByText(t("spaceManager.or"))).toBeInTheDocument();
    await waitFor(() =>
      expect(mockApi.previewSpace).toHaveBeenLastCalledWith(
        expect.objectContaining({
          qbe: [["domain:risk"], ["domain:trading"]],
        })
      )
    );
  });

  test("blocks adding while a term is empty", async () => {
    open();
    startNew();
    const addButton = screen.getByRole("button", {
      name: t("spaceManager.addAlternative"),
    });

    expect(addButton).toBeDisabled();
    await addSelector("domain:trading");
    expect(addButton).toBeEnabled();

    fireEvent.click(addButton);
    expect(addButton).toBeDisabled();
  });

  test("focuses an added alternative", async () => {
    open();
    startNew();
    await addSelector("domain:trading");

    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.addAlternative") })
    );

    const fields = document.querySelectorAll(".tagify__input");
    expect(fields[fields.length - 1]).toHaveFocus();
  });

  test("requires every alternative to carry a tag", async () => {
    open();
    startNew();
    await addSelector("domain:trading");
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.addAlternative") })
    );

    expect(
      screen.getByRole("button", { name: t("spaceManager.next") })
    ).toBeDisabled();
  });

  test("removes an alternative", async () => {
    open();
    startNew();
    await addSelector("domain:trading");
    await addAlternative("domain:risk");
    fireEvent.click(
      screen.getByRole("button", {
        name: `${t("spaceManager.removeAlternative")} 2`,
      })
    );

    await waitFor(() =>
      expect(mockApi.previewSpace).toHaveBeenLastCalledWith(
        expect.objectContaining({ qbe: [["domain:trading"]] })
      )
    );
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
    await nextWhenEnabled();
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.save") })
    );

    await waitFor(() => {
      expect(mockApi.updateSpace).toHaveBeenCalledWith("risk", {
        id: "risk",
        name: "Risk Domain",
        description: "Risk steps",
        qbe: [["domain:risk"]],
      });
      expect(setSpaceId).toHaveBeenCalledWith("risk");
    });
  });

  test("combines added tags into one sorted query", async () => {
    open();
    fireEvent.click(screen.getByText("Risk"));
    await addSelector("domain:trading");
    await addSelector("domain:risk");
    next();
    await nextWhenEnabled();
    fireEvent.click(
      screen.getByRole("button", { name: t("spaceManager.save") })
    );

    await waitFor(() => {
      expect(mockApi.updateSpace).toHaveBeenCalledWith("risk", {
        id: "risk",
        name: "Risk",
        description: "Risk steps",
        qbe: [["domain:risk", "domain:trading"]],
      });
    });
  });

  test("ejects QBE into an editable Lua selector", async () => {
    open();
    fireEvent.click(screen.getByText("Risk"));
    await addSelector("domain:trading");
    next();
    await nextWhenEnabled();

    expect(screen.getByTestId("selector-editor")).toHaveAttribute("readonly");
    editScript();

    expect(screen.getByTestId("selector-editor")).toHaveValue(
      GENERATED_SELECTOR.script
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
    await waitFor(() =>
      expect(
        screen.getByRole("button", { name: t("spaceManager.editScript") })
      ).not.toBeDisabled()
    );
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
    await waitFor(() =>
      expect(
        screen.getByRole("button", { name: t("spaceManager.editScript") })
      ).not.toBeDisabled()
    );
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
    mockApi.previewSpace.mockImplementation(async (space) => ({
      space: { ...space, selector: space.selector ?? GENERATED_SELECTOR },
      step_ids: [],
    }));
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

  test("requires a selector before showing details", () => {
    open();
    startNew();

    expect(
      screen.getByRole("button", { name: t("spaceManager.next") })
    ).toBeDisabled();
  });

  test("accepts a single tag as a selector", async () => {
    open();
    startNew();

    await addSelector("example");
    expect(
      screen.getByRole("button", { name: t("spaceManager.next") })
    ).toBeEnabled();
  });

  test("ignores a blank tag", async () => {
    open();
    startNew();

    await addSelector("   ");
    expect(
      screen.getByRole("button", { name: t("spaceManager.next") })
    ).toBeDisabled();
  });

  test("requires an id and name before saving", async () => {
    open();
    startNew();
    await addSelector("domain:risk");
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
    await nextWhenEnabled();

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
    await addSelector("domain:risk");
    next();
    fireEvent.change(
      screen.getByPlaceholderText("Steps matching domain:risk"),
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

  test("offers the catalog tags as selector suggestions", () => {
    open();
    startNew();

    expect(
      suggestionsOf(screen.getByLabelText(t("stepEditor.tagPlaceholder")))
    ).toEqual(
      expect.arrayContaining(["domain:risk", "domain:trading", "tier:gold"])
    );
  });
});
