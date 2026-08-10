import { render, screen } from "@testing-library/react";
import { I18nProvider, useT } from "./I18nProvider";
import type { MessageValue, Vars } from "./i18nUtils";

const Translation = ({
  messageKey,
  vars,
}: {
  messageKey: string;
  vars?: Vars;
}) => {
  const t = useT();
  return <div>{t(messageKey, vars)}</div>;
};

const renderTranslation = (
  messages: Record<string, MessageValue>,
  messageKey: string,
  vars?: Vars
) =>
  render(
    <I18nProvider messages={messages}>
      <Translation messageKey={messageKey} vars={vars} />
    </I18nProvider>
  );

describe("I18nProvider", () => {
  test.each([
    ["plain text", "Hello World", undefined, "Hello World"],
    ["one variable", "Hello {name}", { name: "Alice" }, "Hello Alice"],
    [
      "several variables",
      "{greeting} {name}, {count} messages",
      { greeting: "Hello", name: "Bob", count: 5 },
      "Hello Bob, 5 messages",
    ],
    [
      "unknown placeholders",
      "Value: {known}, Unknown: {unknown}",
      { known: 123 },
      "Value: 123, Unknown: {unknown}",
    ],
  ])("translates %s", (_, message, vars, expected) => {
    renderTranslation({ key: message }, "key", vars);
    expect(screen.getByText(expected)).toBeInTheDocument();
  });

  test("returns an unknown key", () => {
    renderTranslation({}, "missing.key");
    expect(screen.getByText("missing.key")).toBeInTheDocument();
  });

  test.each([
    [
      "zero",
      { zero: "No items", one: "One item", other: "{count} items" },
      0,
      "No items",
    ],
    [
      "zero fallback",
      { one: "One item", other: "{count} items" },
      0,
      "0 items",
    ],
    [
      "one",
      { zero: "No items", one: "One item", other: "{count} items" },
      1,
      "One item",
    ],
    ["one fallback", { other: "{count} items" }, 1, "1 items"],
    [
      "other",
      { zero: "No items", one: "One item", other: "{count} items" },
      5,
      "5 items",
    ],
    ["large", { other: "{count} items" }, 1_000_000, "1000000 items"],
    [
      "negative",
      { one: "1 credit", other: "{count} credits" },
      -5,
      "-5 credits",
    ],
  ])("selects the %s plural form", (_, message, count, expected) => {
    renderTranslation({ count: message }, "count", { count });
    expect(screen.getByText(expected)).toBeInTheDocument();
  });

  test("interpolates plural variables", () => {
    renderTranslation(
      {
        files: {
          zero: "No files for {name}",
          one: "{name} has 1 file",
          other: "{name} has {count} files",
        },
      },
      "files",
      { count: 3, name: "Alice" }
    );
    expect(screen.getByText("Alice has 3 files")).toBeInTheDocument();
  });

  test.each([[undefined], ["invalid"]])(
    "rejects nonnumeric plural count %p",
    (count) => {
      const warn = jest.spyOn(console, "warn").mockImplementation();
      renderTranslation({ count: { other: "{count} items" } }, "count", {
        ...(count === undefined ? {} : { count }),
      });

      expect(screen.getByText("count")).toBeInTheDocument();
      expect(warn).toHaveBeenCalledWith(
        "Plural message \"count\" requires a numeric 'count' variable"
      );
      warn.mockRestore();
    }
  );

  test.each([
    ["de-CH", ["Kein Step", "1 Step", "5 Steps"]],
    ["it-CH", ["Nessun Step", "1 Step", "5 Steps"]],
    ["fr-CH", ["Aucun Step", "1 Step", "5 Steps"]],
  ])("handles %s zero/one/other forms", (locale, forms) => {
    const messages = {
      steps: { zero: forms[0], one: forms[1], other: "{count} Steps" },
    };
    const { rerender } = render(
      <I18nProvider locale={locale} messages={messages}>
        <Translation messageKey="steps" vars={{ count: 0 }} />
      </I18nProvider>
    );
    expect(screen.getByText(forms[0])).toBeInTheDocument();

    rerender(
      <I18nProvider locale={locale} messages={messages}>
        <Translation messageKey="steps" vars={{ count: 1 }} />
      </I18nProvider>
    );
    expect(screen.getByText(forms[1])).toBeInTheDocument();

    rerender(
      <I18nProvider locale={locale} messages={messages}>
        <Translation messageKey="steps" vars={{ count: 5 }} />
      </I18nProvider>
    );
    expect(screen.getByText("5 Steps")).toBeInTheDocument();
  });

  test("requires the provider", () => {
    const error = jest.spyOn(console, "error").mockImplementation();
    expect(() => render(<Translation messageKey="key" />)).toThrow(
      "useT must be used within an I18nProvider"
    );
    error.mockRestore();
  });
});
