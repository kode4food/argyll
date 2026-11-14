import { fireEvent, render, screen } from "@testing-library/react";
import { useFlowStore } from "@/app/store/flowStore";
import { t } from "@/app/testUtils/i18n";
import StepEditorAttributesSection from "./StepEditorAttributesSection";

describe("StepEditorAttributesSection", () => {
  const attributes = [
    {
      id: "request",
      role: "required" as const,
      name: "request",
    },
    {
      id: "result",
      role: "output" as const,
      name: "result",
    },
  ];
  const baseProps = {
    addAttribute: jest.fn(),
    attributes,
    flowInputOptions: [],
    flowOutputOptions: [],
    removeAttribute: jest.fn(),
    stepType: "service" as const,
    handling: "standard" as const,
    updateAttribute: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
    useFlowStore.setState({ feeds: [] });
  });

  test("shows compensation toggles", () => {
    const { rerender } = render(<StepEditorAttributesSection {...baseProps} />);

    expect(
      screen.queryByRole("button", {
        name: `${t("stepEditor.compensatedAttributeLabel")} request`,
      })
    ).not.toBeInTheDocument();

    rerender(
      <StepEditorAttributesSection {...baseProps} handling="compensated" />
    );
    const compensateButton = screen.getByRole("button", {
      name: `${t("stepEditor.compensatedAttributeLabel")} request`,
    });
    const roleButton = screen.getAllByRole("button", {
      name: t("stepEditor.attrRoleSelect"),
    })[0];

    expect(compensateButton.parentElement).toBe(
      roleButton.parentElement?.parentElement
    );
    fireEvent.click(compensateButton);

    expect(baseProps.updateAttribute).toHaveBeenCalledWith(
      "request",
      "compensated",
      true
    );
  });

  test("says when no Feeds are registered", () => {
    render(
      <StepEditorAttributesSection
        {...baseProps}
        attributes={[{ id: "total", role: "feed" as const, name: "total" }]}
      />
    );

    fireEvent.click(
      screen.getByRole("button", { name: t("stepEditor.feedSelect") })
    );
    expect(
      screen.getByRole("option", { name: t("stepEditor.feedNoneRegistered") })
    ).toBeInTheDocument();
  });

  test("picks the Feed a feed attribute reads", () => {
    const feed = {
      value: "0",
      fold: { language: "argyll", source: "feed" },
      attributes: ["amount"],
    };
    useFlowStore.setState({
      feeds: [
        { ...feed, id: "order-total", name: "Order Total" },
        { ...feed, id: "grand-total", name: "Grand Total" },
      ],
    });
    render(
      <StepEditorAttributesSection
        {...baseProps}
        attributes={[
          {
            id: "total",
            role: "feed" as const,
            name: "total",
            feedId: "order-total",
          },
        ]}
      />
    );

    const picker = screen.getByRole("button", {
      name: t("stepEditor.feedSelect"),
    });
    expect(picker).toHaveTextContent("Order Total (order-total)");
    fireEvent.click(picker);
    fireEvent.click(
      screen.getByRole("option", { name: "Grand Total (grand-total)" })
    );
    expect(baseProps.updateAttribute).toHaveBeenCalledWith(
      "total",
      "feedId",
      "grand-total"
    );
    expect(
      screen.queryByRole("button", {
        name: `${t("stepEditor.mappingLabel")} total`,
      })
    ).not.toBeInTheDocument();
  });

  test("picks a metadata key by label", () => {
    render(
      <StepEditorAttributesSection
        {...baseProps}
        attributes={[
          {
            id: "callback",
            role: "meta" as const,
            name: "callback",
            metaKey: "webhook_url",
          },
        ]}
      />
    );

    const picker = screen.getByRole("button", {
      name: t("stepEditor.metaKeySelect"),
    });
    expect(picker).toHaveTextContent(
      `${t("stepEditor.metaKey.webhook_url")} (webhook_url)`
    );
    expect(screen.getAllByRole("textbox")).toHaveLength(1);

    fireEvent.click(picker);
    fireEvent.click(
      screen.getByRole("option", {
        name: `${t("stepEditor.metaKey.parent_flow_id")} (parent_flow_id)`,
      })
    );
    expect(baseProps.updateAttribute).toHaveBeenCalledWith(
      "callback",
      "metaKey",
      "parent_flow_id"
    );
  });

  test("shows a metric preset by label and selector", () => {
    render(
      <StepEditorAttributesSection
        {...baseProps}
        attributes={[
          {
            id: "took",
            role: "metric" as const,
            name: "took",
            metricLanguage: "argyll",
            metricScript: "elapsed",
          },
        ]}
      />
    );

    const picker = screen.getByRole("button", {
      name: t("stepEditor.metricPresetSelect"),
    });
    expect(picker).toHaveTextContent(
      `${t("stepEditor.metricSelect.elapsed")} (elapsed)`
    );
    expect(screen.getAllByRole("textbox")).toHaveLength(1);
    expect(
      screen.queryByRole("button", {
        name: t("stepEditor.metricLanguageLabel"),
      })
    ).not.toBeInTheDocument();

    fireEvent.click(picker);
    fireEvent.click(
      screen.getByRole("option", {
        name: `${t("stepEditor.metricSelect.work.failed")} (work.failed)`,
      })
    );
    expect(baseProps.updateAttribute).toHaveBeenCalledWith(
      "took",
      "metricLanguage",
      "argyll"
    );
    expect(baseProps.updateAttribute).toHaveBeenCalledWith(
      "took",
      "metricScript",
      "work.failed"
    );
  });

  test("edits a custom metric select", () => {
    render(
      <StepEditorAttributesSection
        {...baseProps}
        attributes={[
          {
            id: "took",
            role: "metric" as const,
            name: "took",
            metricScript: "elapsed",
          },
        ]}
      />
    );

    fireEvent.click(
      screen.getByRole("button", { name: t("stepEditor.metricPresetSelect") })
    );
    fireEvent.click(
      screen.getByRole("option", { name: t("stepEditor.metricSelect.custom") })
    );
    expect(
      screen.getByRole("button", { name: t("stepEditor.metricLanguageLabel") })
    ).toBeInTheDocument();

    const input = screen.getByPlaceholderText(
      t("stepEditor.metricSelectPlaceholder")
    );
    expect(input).toHaveValue("elapsed");
    fireEvent.change(input, { target: { value: "work.attempts.max * 2" } });
    expect(baseProps.updateAttribute).toHaveBeenCalledWith(
      "took",
      "metricScript",
      "work.attempts.max * 2"
    );
  });

  test("opens a non-preset metric select as custom", () => {
    render(
      <StepEditorAttributesSection
        {...baseProps}
        attributes={[
          {
            id: "tries",
            role: "metric" as const,
            name: "tries",
            metricLanguage: "lua",
            metricScript: "return work.attempts.max",
          },
        ]}
      />
    );

    expect(
      screen.getByPlaceholderText(t("stepEditor.metricSelectPlaceholder"))
    ).toHaveValue("return work.attempts.max");
  });

  test("blocks a duplicate selected inner name", () => {
    render(
      <StepEditorAttributesSection
        {...baseProps}
        handling="compensated"
        attributes={[
          { ...attributes[0], mappingName: "value", compensated: true },
          { ...attributes[1], mappingName: "value" },
        ]}
      />
    );

    expect(
      screen.getByRole("button", {
        name: `${t("stepEditor.compensatedAttributeLabel")} result`,
      })
    ).toBeDisabled();
  });
});
