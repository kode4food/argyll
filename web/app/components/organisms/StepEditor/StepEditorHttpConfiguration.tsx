import React from "react";
import DurationInput from "@/app/components/molecules/DurationInput";
import SegmentedGroup from "@/app/components/molecules/SegmentedGroup";
import type { ActionMode, Handling, HTTPMethod, StepType } from "@/app/api";
import { useT } from "@/app/i18n";
import {
  getStepTypeIcon,
  IconActionModeAsync,
  IconActionModeSync,
  IconCompensate,
  IconEndpoint,
  IconHealthCheck,
  type LucideIcon,
} from "@/utils/iconRegistry";
import formStyles from "./StepEditorForm.module.css";
import localStyles from "./StepEditorHttpConfiguration.module.css";
import InlineSelectDropdown, {
  type InlineSelectOption,
} from "./InlineSelectDropdown";
import IconDropdown, { type IconDropdownOption } from "./IconDropdown";

const METHOD_OPTIONS: InlineSelectOption[] = [
  { value: "POST", label: "POST" },
  { value: "GET", label: "GET" },
  { value: "PUT", label: "PUT" },
  { value: "DELETE", label: "DELETE" },
];

const actionModeOptions = (
  t: (key: string) => string
): IconDropdownOption[] => [
  {
    value: "sync",
    label: t("stepEditor.actionMode.sync"),
    icon: <IconActionModeSync className={localStyles.actionModeIcon} />,
  },
  {
    value: "async",
    label: t("stepEditor.actionMode.async"),
    icon: <IconActionModeAsync className={localStyles.actionModeIcon} />,
  },
];

interface ModeControl {
  mode: ActionMode;
  onChange: (value: ActionMode) => void;
}

interface MethodControl {
  method: HTTPMethod;
  onChange: (value: HTTPMethod) => void;
}

interface TimeoutControl {
  value: number;
  onChange: (value: number) => void;
  placeholderMs?: number;
}

interface StepEditorHttpActionFieldsProps {
  mode?: ModeControl;
  method?: MethodControl;
  endpointIcon: LucideIcon;
  endpointLabelKey: string;
  endpointPlaceholderKey: string;
  endpointValue: string;
  onEndpointChange: (value: string) => void;
  timeout?: TimeoutControl;
}

interface StepEditorHttpConfigurationProps {
  endpoint: string;
  httpMethod: HTTPMethod;
  httpMode: ActionMode;
  healthCheck: string;
  compensate: string;
  compensateMethod: HTTPMethod;
  compensateTimeout: number;
  compensateMode: ActionMode;
  httpTimeout: number;
  handling: Handling;
  stepType: StepType;
  setEndpoint: (value: string) => void;
  setHttpMethod: (value: HTTPMethod) => void;
  setHttpMode: (value: ActionMode) => void;
  setHealthCheck: (value: string) => void;
  setCompensate: (value: string) => void;
  setCompensateMethod: (value: HTTPMethod) => void;
  setCompensateTimeout: (value: number) => void;
  setCompensateMode: (value: ActionMode) => void;
  setHttpTimeout: (value: number) => void;
}

const StepEditorHttpActionFields: React.FC<StepEditorHttpActionFieldsProps> = ({
  mode,
  method,
  endpointIcon: EndpointIcon,
  endpointLabelKey,
  endpointPlaceholderKey,
  endpointValue,
  onEndpointChange,
  timeout,
}) => {
  const t = useT();
  const modeOptions = actionModeOptions(t);
  const current = mode?.mode ?? "sync";
  const selectedMode =
    modeOptions.find((o) => o.value === current) ?? modeOptions[0];

  return (
    <div className={formStyles.row}>
      <SegmentedGroup className={formStyles.iconDropdownGroup}>
        <IconDropdown
          ariaLabel={t("stepEditor.actionModeLabel")}
          faceIcon={selectedMode.icon}
          value={current}
          options={mode ? modeOptions : [modeOptions[0]]}
          onChange={(v) => mode?.onChange(v as ActionMode)}
          disabled={!mode}
        />
      </SegmentedGroup>
      <div className={formStyles.fieldNoFlex}>
        <label className={formStyles.label}>
          {t("stepEditor.httpMethodLabel")}
        </label>
        <SegmentedGroup className={localStyles.methodSelect}>
          <InlineSelectDropdown
            value={method?.method ?? "GET"}
            options={method ? METHOD_OPTIONS : [{ value: "GET", label: "GET" }]}
            onChange={(v) => method?.onChange(v as HTTPMethod)}
            disabled={!method}
          />
        </SegmentedGroup>
      </div>
      <div className={`${formStyles.field} ${formStyles.flex1}`}>
        <label className={formStyles.labelWithIcon}>
          <span className={formStyles.labelIcon}>
            <EndpointIcon aria-hidden="true" />
          </span>
          {t(endpointLabelKey)}
        </label>
        <input
          type="text"
          value={endpointValue}
          onChange={(e) => onEndpointChange(e.target.value)}
          placeholder={t(endpointPlaceholderKey)}
          className={formStyles.formControl}
        />
      </div>
      {timeout && (
        <div className={formStyles.fieldNoFlex}>
          <label className={formStyles.label}>
            {t("stepEditor.timeoutLabel")}
          </label>
          <DurationInput
            value={timeout.value}
            onChange={timeout.onChange}
            placeholderMs={timeout.placeholderMs}
          />
        </div>
      )}
    </div>
  );
};

const StepEditorHttpConfiguration: React.FC<
  StepEditorHttpConfigurationProps
> = ({
  endpoint,
  httpMethod,
  httpMode,
  healthCheck,
  compensate,
  compensateMethod,
  compensateTimeout,
  compensateMode,
  httpTimeout,
  handling,
  stepType,
  setEndpoint,
  setHttpMethod,
  setHttpMode,
  setHealthCheck,
  setCompensate,
  setCompensateMethod,
  setCompensateMode,
  setCompensateTimeout,
  setHttpTimeout,
}) => {
  const t = useT();
  const StepTypeIcon = getStepTypeIcon(stepType);

  return (
    <div className={formStyles.section}>
      <div className={formStyles.sectionHeader}>
        <label className={formStyles.labelWithIcon}>
          <span className={formStyles.labelIcon}>
            <StepTypeIcon aria-hidden="true" />
          </span>
          {t("stepEditor.serviceConfigLabel")}
        </label>
      </div>
      <div className={localStyles.httpFields}>
        <StepEditorHttpActionFields
          mode={{ mode: httpMode, onChange: setHttpMode }}
          method={{ method: httpMethod, onChange: setHttpMethod }}
          endpointIcon={IconEndpoint}
          endpointLabelKey="stepEditor.endpointLabel"
          endpointPlaceholderKey="stepEditor.endpointPlaceholder"
          endpointValue={endpoint}
          onEndpointChange={setEndpoint}
          timeout={{ value: httpTimeout, onChange: setHttpTimeout }}
        />
        {handling === "compensated" && (
          <StepEditorHttpActionFields
            mode={{ mode: compensateMode, onChange: setCompensateMode }}
            method={{ method: compensateMethod, onChange: setCompensateMethod }}
            endpointIcon={IconCompensate}
            endpointLabelKey="stepEditor.compensateLabel"
            endpointPlaceholderKey="stepEditor.compensatePlaceholder"
            endpointValue={compensate}
            onEndpointChange={setCompensate}
            timeout={{
              value: compensateTimeout,
              onChange: setCompensateTimeout,
              placeholderMs: httpTimeout,
            }}
          />
        )}
        <StepEditorHttpActionFields
          endpointIcon={IconHealthCheck}
          endpointLabelKey="stepEditor.healthCheckLabel"
          endpointPlaceholderKey="stepEditor.healthCheckPlaceholder"
          endpointValue={healthCheck}
          onEndpointChange={setHealthCheck}
        />
      </div>
    </div>
  );
};

export default StepEditorHttpConfiguration;
