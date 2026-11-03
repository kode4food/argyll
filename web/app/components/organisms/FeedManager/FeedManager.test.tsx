import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { AttributeRole, Feed, Step } from "@/app/api";
import { useFlowStore } from "@/app/store/flowStore";
import { t } from "@/app/testUtils/i18n";
import { addTag, suggestionsOf } from "@/app/testUtils/tags";
import FeedManager from "./FeedManager";

jest.mock("@/app/components/molecules/ScriptEditor", () => ({
  __esModule: true,
  default: ({ value, onChange, language }: any) => (
    <textarea
      data-testid="fold-editor"
      data-language={language}
      value={value}
      onChange={(e) => onChange(e.target.value)}
    />
  ),
}));

jest.mock("@/app/api", () => ({
  ...jest.requireActual("@/app/api"),
  api: {
    getFeed: jest.fn(),
    registerFeed: jest.fn(),
    updateFeed: jest.fn(),
    unregisterFeed: jest.fn(),
  },
}));

import { api } from "@/app/api";

const mockApi = api as jest.Mocked<typeof api>;

const TOTAL: Feed = {
  id: "total",
  name: "Order Total",
  value: "0",
  fold: { language: "argyll", script: "feed + amount" },
  attributes: ["amount"],
};

const BILLING = { id: "billing", name: "Billing" };

const STEPS: Step[] = [
  {
    id: "charge",
    name: "Charge",
    type: "service",
    attributes: {
      amount: { role: AttributeRole.Output },
      order: { role: AttributeRole.Required },
    },
  },
  {
    id: "notify",
    name: "Notify",
    type: "service",
    attributes: { order: { role: AttributeRole.Required } },
  },
];

const button = (key: string) => screen.getByRole("button", { name: t(key) });

const choose = (labelKey: string, option: string) => {
  fireEvent.click(screen.getByRole("button", { name: t(labelKey) }));
  fireEvent.click(screen.getByRole("option", { name: option }));
};

const typeInto = (placeholderKey: string, value: string) =>
  fireEvent.change(screen.getByPlaceholderText(t(placeholderKey)), {
    target: { value },
  });

const attributesInput = () =>
  screen.getByLabelText(t("feedManager.attributesPlaceholder"));

describe("FeedManager", () => {
  beforeEach(() => {
    jest.resetAllMocks();
    useFlowStore.setState({
      feeds: [TOTAL],
      steps: STEPS,
      spaces: [BILLING],
      watchedFeed: null,
    });
    mockApi.getFeed.mockResolvedValue({
      feed: TOTAL,
      state: {
        id: "total",
        version: "v1",
        value: 42,
        last_updated: "2024-01-01T00:00:00Z",
      },
    });
    mockApi.registerFeed.mockImplementation(async (feed) => feed);
    mockApi.updateFeed.mockImplementation(async (_, feed) => feed);
    mockApi.unregisterFeed.mockResolvedValue();
  });

  const open = (onClose = jest.fn()) =>
    render(<FeedManager isOpen onClose={onClose} />);

  const openTotal = async () => {
    open();
    fireEvent.click(screen.getByText("Order Total"));
    await screen.findByText("42");
  };

  test("lists Feeds", () => {
    open();
    expect(screen.getByText("Order Total")).toBeInTheDocument();
    expect(screen.getByText("total")).toBeInTheDocument();
  });

  test("shows empty state", () => {
    useFlowStore.setState({ feeds: [] });
    open();
    expect(screen.getByText(t("feedManager.empty"))).toBeInTheDocument();
  });

  test("creates a Feed", async () => {
    open();
    fireEvent.click(button("feedManager.new"));
    expect(button("feedManager.save")).toBeDisabled();

    typeInto("feedManager.idPlaceholder", "count");
    typeInto("feedManager.namePlaceholder", "Count");
    await addTag(attributesInput(), "amount");
    choose("feedManager.spaceLabel", "Billing");
    fireEvent.change(screen.getByTestId("fold-editor"), {
      target: { value: "feed + 1" },
    });
    fireEvent.click(button("feedManager.save"));

    await waitFor(() =>
      expect(mockApi.registerFeed).toHaveBeenCalledWith({
        id: "count",
        name: "Count",
        value: "0",
        fold: { language: "argyll", script: "feed + 1" },
        attributes: ["amount"],
        space_id: "billing",
      })
    );
    await waitFor(() => expect(button("feedManager.new")).toBeInTheDocument());
  });

  test("offers and accepts only declared Step outputs", async () => {
    open();
    fireEvent.click(button("feedManager.new"));
    expect(suggestionsOf(attributesInput())).toEqual(["amount"]);

    await addTag(attributesInput(), "order");
    typeInto("feedManager.idPlaceholder", "count");
    typeInto("feedManager.namePlaceholder", "Count");
    fireEvent.change(screen.getByTestId("fold-editor"), {
      target: { value: "feed" },
    });
    expect(button("feedManager.save")).toBeDisabled();
  });

  test("rejects an invalid seed", async () => {
    await openTotal();
    typeInto("feedManager.seedPlaceholder", "{nope");
    expect(button("feedManager.save")).toBeDisabled();
  });

  test("edits a Feed and shows its current value", async () => {
    await openTotal();
    expect(mockApi.getFeed).toHaveBeenCalledWith("total");
    expect(screen.getByDisplayValue("total")).toBeDisabled();

    typeInto("feedManager.namePlaceholder", "Grand Total");
    fireEvent.click(button("feedManager.save"));

    await waitFor(() =>
      expect(mockApi.updateFeed).toHaveBeenCalledWith("total", {
        ...TOTAL,
        name: "Grand Total",
      })
    );
    expect(useFlowStore.getState().watchedFeed).toBeNull();
  });

  test("loads a Feed's Attributes as tags", async () => {
    await openTotal();
    expect(screen.getByTitle("amount")).toBeInTheDocument();
    expect(button("feedManager.save")).toBeEnabled();
  });

  test("deletes a Feed", async () => {
    await openTotal();
    fireEvent.click(button("feedManager.delete"));
    await waitFor(() =>
      expect(mockApi.unregisterFeed).toHaveBeenCalledWith("total")
    );
  });

  test("shows save errors", async () => {
    mockApi.updateFeed.mockRejectedValue(new Error("boom"));
    await openTotal();
    fireEvent.click(button("feedManager.save"));
    expect(await screen.findByText("boom")).toBeInTheDocument();
  });

  test("shows load errors", async () => {
    mockApi.getFeed.mockRejectedValue(new Error(""));
    open();
    fireEvent.click(screen.getByText("Order Total"));
    expect(
      await screen.findByText(t("feedManager.loadFailed"))
    ).toBeInTheDocument();
  });

  test("ignores a Feed read that finishes after close", async () => {
    let finish!: (res: Awaited<ReturnType<typeof api.getFeed>>) => void;
    mockApi.getFeed.mockReturnValueOnce(
      new Promise((resolve) => {
        finish = resolve;
      })
    );
    open();
    fireEvent.click(screen.getByText("Order Total"));
    fireEvent.click(button("feedManager.cancel"));
    await act(async () => {
      finish({
        feed: TOTAL,
        state: { id: TOTAL.id, version: "v1", value: 42, last_updated: "" },
      });
    });
    expect(useFlowStore.getState().watchedFeed).toBeNull();
  });

  test("closes and stops watching", async () => {
    const onClose = jest.fn();
    render(<FeedManager isOpen onClose={onClose} />);
    fireEvent.click(screen.getByText("Order Total"));
    await screen.findByText("42");
    fireEvent.click(button("feedManager.cancel"));
    fireEvent.click(button("feedManager.close"));
    expect(onClose).toHaveBeenCalled();
    expect(useFlowStore.getState().watchedFeed).toBeNull();
  });
});
