import { AttributeRole, Feed } from "@/app/api";
import {
  emptyDraft,
  foldCompletions,
  isDraftValid,
  outputNames,
  seedDraft,
  toDraft,
  toFeed,
} from "./feedManagerUtils";

const FEED: Feed = {
  id: "total",
  name: "Total",
  description: "Running total",
  value: "0",
  fold: { language: "lua", source: "return feed + amount" },
  attributes: ["amount"],
  space_id: "billing",
};

describe("feedManagerUtils", () => {
  test("round trips a Feed through a draft", () => {
    expect(toFeed(toDraft(FEED))).toEqual(FEED);
  });

  test("seeds a new Feed from what a chat draft gives", () => {
    expect(seedDraft({})).toEqual(emptyDraft());
    expect(toFeed(seedDraft(FEED))).toEqual(FEED);
    const nulls = { fold: null, attributes: null } as unknown as Partial<Feed>;
    expect(seedDraft(nulls)).toEqual(emptyDraft());
  });

  test("trims and drops an empty description and Space", () => {
    const draft = {
      ...toDraft(FEED),
      id: " total ",
      description: " ",
      spaceId: "",
    };
    const feed = toFeed(draft);
    expect(feed.id).toBe("total");
    expect("description" in feed).toBe(false);
    expect("space_id" in feed).toBe(false);
  });

  test.each([
    ["complete", {}, true],
    ["missing ID", { id: " " }, false],
    ["missing name", { name: "" }, false],
    ["invalid seed", { value: "{" }, false],
    ["empty seed", { value: " " }, false],
    ["empty fold", { fold: { language: "lua", source: " " } }, false],
    ["no attributes", { attributes: [] }, false],
    ["reserved attribute", { attributes: ["feed"] }, false],
  ])("validates a %s draft", (_, patch, expected) => {
    expect(isDraftValid({ ...toDraft(FEED), ...patch })).toBe(expected);
  });

  test("starts with an invalid empty draft", () => {
    expect(isDraftValid(emptyDraft())).toBe(false);
  });

  test("lists every Step output and metric once", () => {
    expect(
      outputNames([
        {
          id: "s",
          name: "S",
          type: "service",
          attributes: {
            b: { role: AttributeRole.Output },
            a: { role: AttributeRole.Output },
            c: { role: AttributeRole.Required },
            d: {
              role: AttributeRole.Metric,
              metric: {
                selector: { language: "argyll", source: "elapsed" },
              },
            },
          },
        },
        {
          id: "t",
          name: "T",
          type: "service",
          attributes: { a: { role: AttributeRole.Output } },
        },
      ])
    ).toEqual(["a", "b", "d"]);
  });

  test("completes the fold bindings", () => {
    const draft = { ...toDraft(FEED), attributes: ["amount", "refunded"] };
    expect(foldCompletions(draft)).toEqual(["feed", "amount", "refunded"]);
  });
});
