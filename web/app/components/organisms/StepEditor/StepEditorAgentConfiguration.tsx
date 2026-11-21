import React from "react";
import type { AgentConfig, MCPServer } from "@/app/api";
import DurationInput from "@/app/components/molecules/DurationInput";
import ScriptEditor from "@/app/components/molecules/ScriptEditor";
import SelectField, {
  type SelectFieldOption,
} from "@/app/components/molecules/SelectField";
import TagInput from "@/app/components/molecules/TagInput";
import { useT } from "@/app/i18n";
import {
  IconAdd,
  IconMCPServer,
  IconPrologue,
  IconRemove,
  IconStepTypeAgent,
} from "@/utils/iconRegistry";
import styles from "./StepEditor.module.css";
import formStyles from "./StepEditorForm.module.css";
import localStyles from "./StepEditorAgentConfiguration.module.css";
import { useAgentCapabilities } from "./useAgentCapabilities";

interface StepEditorAgentConfigurationProps {
  agent: AgentConfig;
  setAgent: (agent: AgentConfig) => void;
}

interface AgentNumberFieldProps {
  label: string;
  value: number;
  onChange: (value: number) => void;
}

const NO_TOOLS: readonly string[] = [];

const options = (names: string[]): SelectFieldOption[] =>
  names.map((name) => ({ label: name, value: name }));

const effortLabel = (effort: string): string => {
  if (effort === "xhigh") return "Extra High";
  return effort
    .split(/[-_\s]+/)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
};

const offeredTools = (
  offered: MCPServer[],
  server: string
): readonly string[] =>
  offered.find((s) => s.server === server)?.tools ?? NO_TOOLS;

const AgentNumberField: React.FC<AgentNumberFieldProps> = ({
  label,
  value,
  onChange,
}) => (
  <div className={formStyles.fieldNoFlex}>
    <label className={formStyles.label}>{label}</label>
    <input
      type="number"
      min={1}
      value={value || ""}
      onChange={(e) => onChange(Number(e.target.value))}
      className={localStyles.numberInput}
    />
  </div>
);

const StepEditorAgentConfiguration: React.FC<
  StepEditorAgentConfigurationProps
> = ({ agent, setAgent }) => {
  const t = useT();
  const { providers, problems = [] } = useAgentCapabilities();
  const provider = providers.find((item) => item.provider === agent.provider);
  const models = provider?.models ?? [];
  const efforts = models.find(({ id }) => id === agent.model)?.efforts;
  const defaultEffort = t("stepEditor.agentEffortDefault");
  const offered = provider?.mcp ?? [];
  const servers: MCPServer[] = agent.mcp?.length ? agent.mcp : [{ server: "" }];
  const available = offered.filter(
    (item) => !servers.some((s) => s.server === item.server)
  );
  const update = (patch: Partial<AgentConfig>) =>
    setAgent({ ...agent, ...patch });
  const updateServer = (idx: number, patch: Partial<MCPServer>) =>
    update({
      mcp: servers.map((s, i) => (i === idx ? { ...s, ...patch } : s)),
    });

  return (
    <>
      <section
        aria-label={t("stepEditor.agentConfigLabel")}
        className={formStyles.section}
      >
        <label className={formStyles.labelWithIcon}>
          <span className={formStyles.labelIcon}>
            <IconStepTypeAgent aria-hidden="true" />
          </span>
          {t("stepEditor.agentConfigLabel")}
        </label>
        {problems.length > 0 && (
          <div role="alert" className={formStyles.errorMessage}>
            {problems.map((problem, idx) => (
              <p key={`${problem.instance ?? "discovery"}-${idx}`}>
                {problem.instance && <strong>{problem.instance}: </strong>}
                {problem.status && <span>({problem.status}) </span>}
                {problem.detail ||
                  problem.title ||
                  t("stepEditor.agentDiscoveryFailed")}
              </p>
            ))}
          </div>
        )}
        <div className={localStyles.settingsRow}>
          <div className={localStyles.providerField}>
            <label className={formStyles.label}>
              {t("stepEditor.agentProviderLabel")}
            </label>
            <SelectField
              ariaLabel={t("stepEditor.agentProviderLabel")}
              value={agent.provider}
              onChange={(name) => {
                if (name !== agent.provider) {
                  update({
                    provider: name,
                    model: "",
                    effort: undefined,
                    mcp: [],
                  });
                }
              }}
              options={options(providers.map((item) => item.provider))}
              placeholder={t("stepEditor.agentProviderPlaceholder")}
            />
          </div>
          <div className={localStyles.modelField}>
            <label className={formStyles.label}>
              {t("stepEditor.agentModelLabel")}
            </label>
            <SelectField
              ariaLabel={t("stepEditor.agentModelLabel")}
              value={agent.model}
              onChange={(model) => {
                if (model !== agent.model) update({ model, effort: undefined });
              }}
              options={options(models.map(({ id }) => id))}
              placeholder={t("stepEditor.agentModelPlaceholder")}
            />
          </div>
          <div className={localStyles.providerField}>
            <label className={formStyles.label}>
              {t("stepEditor.agentEffortLabel")}
            </label>
            <SelectField
              ariaLabel={t("stepEditor.agentEffortLabel")}
              value={agent.effort ?? ""}
              onChange={(effort) => update({ effort: effort || undefined })}
              options={
                efforts?.length
                  ? [
                      { label: defaultEffort, value: "" },
                      ...efforts.map((value) => ({
                        label: effortLabel(value),
                        value,
                      })),
                    ]
                  : []
              }
              placeholder={agent.effort || defaultEffort}
            />
          </div>
          <AgentNumberField
            label={t("stepEditor.agentMaxTurnsLabel")}
            value={agent.max_turns}
            onChange={(max_turns) => update({ max_turns })}
          />
          <AgentNumberField
            label={t("stepEditor.agentMaxTokensLabel")}
            value={agent.max_tokens}
            onChange={(max_tokens) => update({ max_tokens })}
          />
          <div className={formStyles.fieldNoFlex}>
            <label className={formStyles.label}>
              {t("stepEditor.timeoutLabel")}
            </label>
            <DurationInput
              value={agent.timeout ?? 0}
              onChange={(timeout) => update({ timeout })}
            />
          </div>
        </div>
      </section>
      <section
        aria-label={t("stepEditor.agentMCPLabel")}
        className={formStyles.section}
      >
        <div className={formStyles.sectionHeader}>
          <label className={formStyles.labelWithIcon}>
            <span className={formStyles.labelIcon}>
              <IconMCPServer aria-hidden="true" />
            </span>
            {t("stepEditor.agentMCPLabel")}
          </label>
          <button
            type="button"
            disabled={available.length === 0 || servers.some((s) => !s.server)}
            onClick={() => update({ mcp: [...servers, { server: "" }] })}
            className={`${formStyles.addButtonStyle} ${localStyles.addServer}`}
            title={t("stepEditor.addMCPServer")}
          >
            <IconAdd className={styles.iconMd} />
          </button>
        </div>
        <div className={formStyles.argList}>
          {servers.map((s, idx) => (
            <div key={idx} className={formStyles.attrRowInputs}>
              <SelectField
                ariaLabel={t("stepEditor.agentMCPLabel")}
                className={localStyles.mcpServerField}
                Icon={IconMCPServer}
                onChange={(server) => {
                  if (server !== s.server) {
                    updateServer(idx, { server, tools: undefined });
                  }
                }}
                options={options(
                  offered
                    .filter(
                      (item) =>
                        item.server === s.server || available.includes(item)
                    )
                    .map((item) => item.server)
                )}
                placeholder={t("stepEditor.agentMCPServerSelect")}
                value={s.server}
              />
              <div
                className={`${formStyles.field} ${formStyles.flex2}`}
                inert={!s.server}
              >
                <TagInput
                  Icon={IconMCPServer}
                  readOnly={!s.server}
                  removeLabel={t("stepEditor.removeMCPTool")}
                  placeholder={t(
                    s.tools?.length === 0
                      ? "stepEditor.agentMCPNoTools"
                      : "stepEditor.agentMCPToolsPlaceholder"
                  )}
                  tags={s.tools ?? []}
                  onChange={(tools) => updateServer(idx, { tools })}
                  shouldEnforceSuggestions={
                    offeredTools(offered, s.server) !== NO_TOOLS
                  }
                  suggestions={offeredTools(offered, s.server)}
                />
              </div>
              {(s.server || servers.length > 1) && (
                <button
                  onClick={() =>
                    update({ mcp: servers.filter((_, i) => i !== idx) })
                  }
                  className={`${formStyles.iconButton} ${formStyles.removeButtonStyle}`}
                  title={t("stepEditor.removeMCPServer")}
                >
                  <IconRemove className={styles.iconSm} />
                </button>
              )}
            </div>
          ))}
        </div>
      </section>
      <section
        aria-label={t("stepEditor.agentPrologueLabel")}
        className={formStyles.section}
      >
        <label className={formStyles.labelWithIcon}>
          <span className={formStyles.labelIcon}>
            <IconPrologue aria-hidden="true" />
          </span>
          {t("stepEditor.agentPrologueLabel")}
        </label>
        <div className={localStyles.prologueEditor}>
          <ScriptEditor
            value={agent.prologue}
            onChange={(prologue) => update({ prologue })}
            language="markdown"
          />
        </div>
      </section>
    </>
  );
};

export default StepEditorAgentConfiguration;
