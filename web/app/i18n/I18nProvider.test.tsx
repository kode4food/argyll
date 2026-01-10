import { render, screen } from "@testing-library/react";
import { I18nProvider, useT } from "./I18nProvider";

const TestComponent = ({
  messageKey,
  vars,
}: {
  messageKey: string;
  vars?: Record<string, string | number>;
}) => {
  const t = useT();
  return <div>{t(messageKey, vars)}</div>;
};

describe("I18nProvider", () => {
  describe("simple string messages", () => {
    it("renders simple message without variables", () => {
      const messages = {
        "test.simple": "Hello World",
      };

      render(
        <I18nProvider locale="en-US" messages={messages}>
          <TestComponent messageKey="test.simple" />
        </I18nProvider>
      );

      expect(screen.getByText("Hello World")).toBeInTheDocument();
    });

    it("interpolates single variable", () => {
      const messages = {
        "test.greeting": "Hello {name}",
      };

      render(
        <I18nProvider locale="en-US" messages={messages}>
          <TestComponent messageKey="test.greeting" vars={{ name: "Alice" }} />
        </I18nProvider>
      );

      expect(screen.getByText("Hello Alice")).toBeInTheDocument();
    });

    it("interpolates multiple variables", () => {
      const messages = {
        "test.multiple": "{greeting} {name}, you have {count} messages",
      };

      render(
        <I18nProvider locale="en-US" messages={messages}>
          <TestComponent
            messageKey="test.multiple"
            vars={{ greeting: "Hello", name: "Bob", count: 5 }}
          />
        </I18nProvider>
      );

      expect(
        screen.getByText("Hello Bob, you have 5 messages")
      ).toBeInTheDocument();
    });

    it("preserves unknown placeholders", () => {
      const messages = {
        "test.unknown": "Value: {known}, Unknown: {unknown}",
      };

      render(
        <I18nProvider locale="en-US" messages={messages}>
          <TestComponent messageKey="test.unknown" vars={{ known: "123" }} />
        </I18nProvider>
      );

      expect(
        screen.getByText("Value: 123, Unknown: {unknown}")
      ).toBeInTheDocument();
    });

    it("returns key when message not found", () => {
      const messages = {};

      render(
        <I18nProvider locale="en-US" messages={messages}>
          <TestComponent messageKey="missing.key" />
        </I18nProvider>
      );

      expect(screen.getByText("missing.key")).toBeInTheDocument();
    });
  });

  describe("plural forms", () => {
    describe("zero form", () => {
      it("uses zero form when count is 0 and zero is defined", () => {
        const messages = {
          "items.count": {
            zero: "No items",
            one: "One item",
            other: "{count} items",
          },
        };

        render(
          <I18nProvider locale="en-US" messages={messages}>
            <TestComponent messageKey="items.count" vars={{ count: 0 }} />
          </I18nProvider>
        );

        expect(screen.getByText("No items")).toBeInTheDocument();
      });

      it("falls back to other form when count is 0 and zero is undefined", () => {
        const messages = {
          "items.count": {
            one: "One item",
            other: "{count} items",
          },
        };

        render(
          <I18nProvider locale="en-US" messages={messages}>
            <TestComponent messageKey="items.count" vars={{ count: 0 }} />
          </I18nProvider>
        );

        expect(screen.getByText("0 items")).toBeInTheDocument();
      });
    });

    describe("one form", () => {
      it("uses one form when count is 1 and one is defined", () => {
        const messages = {
          "items.count": {
            zero: "No items",
            one: "One item",
            other: "{count} items",
          },
        };

        render(
          <I18nProvider locale="en-US" messages={messages}>
            <TestComponent messageKey="items.count" vars={{ count: 1 }} />
          </I18nProvider>
        );

        expect(screen.getByText("One item")).toBeInTheDocument();
      });

      it("falls back to other form when count is 1 and one is undefined", () => {
        const messages = {
          "items.count": {
            other: "{count} items",
          },
        };

        render(
          <I18nProvider locale="en-US" messages={messages}>
            <TestComponent messageKey="items.count" vars={{ count: 1 }} />
          </I18nProvider>
        );

        expect(screen.getByText("1 items")).toBeInTheDocument();
      });
    });

    describe("other form", () => {
      it("uses other form for count > 1", () => {
        const messages = {
          "items.count": {
            zero: "No items",
            one: "One item",
            other: "{count} items",
          },
        };

        render(
          <I18nProvider locale="en-US" messages={messages}>
            <TestComponent messageKey="items.count" vars={{ count: 5 }} />
          </I18nProvider>
        );

        expect(screen.getByText("5 items")).toBeInTheDocument();
      });

      it("interpolates variables in plural forms", () => {
        const messages = {
          "user.files": {
            zero: "No files for {name}",
            one: "{name} has 1 file",
            other: "{name} has {count} files",
          },
        };

        render(
          <I18nProvider locale="en-US" messages={messages}>
            <TestComponent
              messageKey="user.files"
              vars={{ count: 3, name: "Alice" }}
            />
          </I18nProvider>
        );

        expect(screen.getByText("Alice has 3 files")).toBeInTheDocument();
      });
    });

    describe("error handling", () => {
      it("warns and returns key when plural message lacks count variable", () => {
        const consoleWarnSpy = jest.spyOn(console, "warn").mockImplementation();

        const messages = {
          "items.count": {
            zero: "No items",
            one: "One item",
            other: "{count} items",
          },
        };

        render(
          <I18nProvider locale="en-US" messages={messages}>
            <TestComponent messageKey="items.count" vars={{}} />
          </I18nProvider>
        );

        expect(screen.getByText("items.count")).toBeInTheDocument();
        expect(consoleWarnSpy).toHaveBeenCalledWith(
          "Plural message \"items.count\" requires a numeric 'count' variable"
        );

        consoleWarnSpy.mockRestore();
      });

      it("warns and returns key when count is not a number", () => {
        const consoleWarnSpy = jest.spyOn(console, "warn").mockImplementation();

        const messages = {
          "items.count": {
            zero: "No items",
            one: "One item",
            other: "{count} items",
          },
        };

        render(
          <I18nProvider locale="en-US" messages={messages}>
            <TestComponent
              messageKey="items.count"
              vars={{ count: "invalid" as any }}
            />
          </I18nProvider>
        );

        expect(screen.getByText("items.count")).toBeInTheDocument();
        expect(consoleWarnSpy).toHaveBeenCalledWith(
          "Plural message \"items.count\" requires a numeric 'count' variable"
        );

        consoleWarnSpy.mockRestore();
      });
    });
  });

  describe("real-world examples", () => {
    it("handles German zero/one/other distinction", () => {
      const messages = {
        "steps.registered": {
          zero: "Kein Step registriert",
          one: "1 Step registriert",
          other: "{count} Steps registriert",
        },
      };

      const { rerender } = render(
        <I18nProvider locale="de-CH" messages={messages}>
          <TestComponent messageKey="steps.registered" vars={{ count: 0 }} />
        </I18nProvider>
      );

      expect(screen.getByText("Kein Step registriert")).toBeInTheDocument();

      rerender(
        <I18nProvider locale="de-CH" messages={messages}>
          <TestComponent messageKey="steps.registered" vars={{ count: 1 }} />
        </I18nProvider>
      );

      expect(screen.getByText("1 Step registriert")).toBeInTheDocument();

      rerender(
        <I18nProvider locale="de-CH" messages={messages}>
          <TestComponent messageKey="steps.registered" vars={{ count: 5 }} />
        </I18nProvider>
      );

      expect(screen.getByText("5 Steps registriert")).toBeInTheDocument();
    });

    it("handles Italian singular/plural agreement", () => {
      const messages = {
        "steps.registered": {
          zero: "Nessun Step registrato",
          one: "1 Step registrato",
          other: "{count} Steps registrati",
        },
      };

      const { rerender } = render(
        <I18nProvider locale="it-CH" messages={messages}>
          <TestComponent messageKey="steps.registered" vars={{ count: 0 }} />
        </I18nProvider>
      );

      expect(screen.getByText("Nessun Step registrato")).toBeInTheDocument();

      rerender(
        <I18nProvider locale="it-CH" messages={messages}>
          <TestComponent messageKey="steps.registered" vars={{ count: 1 }} />
        </I18nProvider>
      );

      expect(screen.getByText("1 Step registrato")).toBeInTheDocument();

      rerender(
        <I18nProvider locale="it-CH" messages={messages}>
          <TestComponent messageKey="steps.registered" vars={{ count: 10 }} />
        </I18nProvider>
      );

      expect(screen.getByText("10 Steps registrati")).toBeInTheDocument();
    });

    it("handles French singular/plural agreement", () => {
      const messages = {
        "steps.registered": {
          zero: "Aucun Step enregistré",
          one: "1 Step enregistré",
          other: "{count} Steps enregistrés",
        },
      };

      const { rerender } = render(
        <I18nProvider locale="fr-CH" messages={messages}>
          <TestComponent messageKey="steps.registered" vars={{ count: 0 }} />
        </I18nProvider>
      );

      expect(screen.getByText("Aucun Step enregistré")).toBeInTheDocument();

      rerender(
        <I18nProvider locale="fr-CH" messages={messages}>
          <TestComponent messageKey="steps.registered" vars={{ count: 1 }} />
        </I18nProvider>
      );

      expect(screen.getByText("1 Step enregistré")).toBeInTheDocument();

      rerender(
        <I18nProvider locale="fr-CH" messages={messages}>
          <TestComponent messageKey="steps.registered" vars={{ count: 3 }} />
        </I18nProvider>
      );

      expect(screen.getByText("3 Steps enregistrés")).toBeInTheDocument();
    });

    it("handles edge cases with large numbers", () => {
      const messages = {
        "items.count": {
          zero: "No items",
          one: "One item",
          other: "{count} items",
        },
      };

      render(
        <I18nProvider locale="en-US" messages={messages}>
          <TestComponent messageKey="items.count" vars={{ count: 1000000 }} />
        </I18nProvider>
      );

      expect(screen.getByText("1000000 items")).toBeInTheDocument();
    });

    it("handles negative numbers", () => {
      const messages = {
        balance: {
          one: "1 credit",
          other: "{count} credits",
        },
      };

      render(
        <I18nProvider locale="en-US" messages={messages}>
          <TestComponent messageKey="balance" vars={{ count: -5 }} />
        </I18nProvider>
      );

      expect(screen.getByText("-5 credits")).toBeInTheDocument();
    });
  });

  describe("useT hook", () => {
    it("throws error when used outside I18nProvider", () => {
      const consoleErrorSpy = jest.spyOn(console, "error").mockImplementation();

      expect(() => {
        render(<TestComponent messageKey="test.key" />);
      }).toThrow("useT must be used within an I18nProvider");

      consoleErrorSpy.mockRestore();
    });
  });
});
