import React from "react";
import DurationInput from "@/app/components/molecules/DurationInput";
import { HTTPMethod } from "@/app/api";
import { useT } from "@/app/i18n";
import {
  IconCompensate,
  IconEndpoint,
  IconHealthCheck,
} from "@/utils/iconRegistry";
import SegmentedGroup from "@/app/components/molecules/SegmentedGroup";
import formStyles from "./StepEditorForm.module.css";
import localStyles from "./StepEditorHttpConfiguration.module.css";
import InlineSelectDropdown from "./InlineSelectDropdown";

interface StepEditorHttpConfigurationProps {
  endpoint: string;
  httpMethod: HTTPMethod;
  healthCheck: string;
  compensate: string;
  compensateMethod: HTTPMethod;
  compensateTimeout: number;
  httpTimeout: number;
  memoizable: boolean;
  setEndpoint: (value: string) => void;
  setHttpMethod: (value: HTTPMethod) => void;
  setHealthCheck: (value: string) => void;
  setCompensate: (value: string) => void;
  setCompensateMethod: (value: HTTPMethod) => void;
  setCompensateTimeout: (value: number) => void;
  setHttpTimeout: (value: number) => void;
}

const methodOptions = [
  { value: "POST", label: "POST" },
  { value: "GET", label: "GET" },
  { value: "PUT", label: "PUT" },
  { value: "DELETE", label: "DELETE" },
];

const StepEditorHttpConfiguration: React.FC<
  StepEditorHttpConfigurationProps
> = ({
  endpoint,
  httpMethod,
  healthCheck,
  compensate,
  compensateMethod,
  compensateTimeout,
  httpTimeout,
  memoizable,
  setEndpoint,
  setHttpMethod,
  setHealthCheck,
  setCompensate,
  setCompensateMethod,
  setCompensateTimeout,
  setHttpTimeout,
}) => {
  const t = useT();

  return (
    <div className={formStyles.section}>
      <div className={formStyles.sectionHeader}>
        <label className={formStyles.label}>
          {t("stepEditor.httpConfigLabel")}
        </label>
      </div>
      <div className={localStyles.httpFields}>
        <div className={formStyles.row}>
          <div className={formStyles.fieldNoFlex}>
            <label className={formStyles.label}>
              {t("stepEditor.httpMethodLabel")}
            </label>
            <SegmentedGroup className={localStyles.methodSelect}>
              <InlineSelectDropdown
                value={httpMethod}
                options={methodOptions}
                onChange={(v) => setHttpMethod(v as HTTPMethod)}
              />
            </SegmentedGroup>
          </div>
          <div className={`${formStyles.field} ${formStyles.flex1}`}>
            <label className={formStyles.labelWithIcon}>
              <span className={formStyles.labelIcon}>
                <IconEndpoint aria-hidden="true" />
              </span>
              {t("stepEditor.endpointLabel")}
            </label>
            <input
              type="text"
              value={endpoint}
              onChange={(e) => setEndpoint(e.target.value)}
              placeholder={t("stepEditor.endpointPlaceholder")}
              className={formStyles.formControl}
            />
          </div>
          <div className={formStyles.fieldNoFlex}>
            <label className={formStyles.label}>
              {t("stepEditor.timeoutLabel")}
            </label>
            <DurationInput value={httpTimeout} onChange={setHttpTimeout} />
          </div>
        </div>
        <div className={formStyles.row}>
          <div className={formStyles.fieldNoFlex}>
            <label className={formStyles.label}>
              {t("stepEditor.httpMethodLabel")}
            </label>
            <SegmentedGroup className={localStyles.methodSelect}>
              <InlineSelectDropdown
                value={compensateMethod}
                options={methodOptions}
                onChange={(v) => setCompensateMethod(v as HTTPMethod)}
                disabled={memoizable}
              />
            </SegmentedGroup>
          </div>
          <div className={`${formStyles.field} ${formStyles.flex1}`}>
            <label className={formStyles.labelWithIcon}>
              <span className={formStyles.labelIcon}>
                <IconCompensate aria-hidden="true" />
              </span>
              {t("stepEditor.compensateLabel")}
            </label>
            <input
              type="text"
              value={compensate}
              onChange={(e) => setCompensate(e.target.value)}
              placeholder={t("stepEditor.compensatePlaceholder")}
              className={formStyles.formControl}
              disabled={memoizable}
              title={
                memoizable
                  ? t("stepEditor.compensateDisabledMemoizable")
                  : undefined
              }
            />
          </div>
          <div className={formStyles.fieldNoFlex}>
            <label className={formStyles.label}>
              {t("stepEditor.timeoutLabel")}
            </label>
            <DurationInput
              value={compensateTimeout}
              onChange={setCompensateTimeout}
              placeholderMs={httpTimeout}
            />
          </div>
        </div>
        <div className={formStyles.row}>
          <div className={formStyles.fieldNoFlex}>
            <label className={formStyles.label}>
              {t("stepEditor.httpMethodLabel")}
            </label>
            <SegmentedGroup className={localStyles.methodSelect}>
              <InlineSelectDropdown
                value="GET"
                options={[{ value: "GET", label: "GET" }]}
                onChange={() => {}}
                disabled
              />
            </SegmentedGroup>
          </div>
          <div className={`${formStyles.field} ${formStyles.flex1}`}>
            <label className={formStyles.labelWithIcon}>
              <span className={formStyles.labelIcon}>
                <IconHealthCheck aria-hidden="true" />
              </span>
              {t("stepEditor.healthCheckLabel")}
            </label>
            <input
              type="text"
              value={healthCheck}
              onChange={(e) => setHealthCheck(e.target.value)}
              placeholder={t("stepEditor.healthCheckPlaceholder")}
              className={formStyles.formControl}
            />
          </div>
        </div>
      </div>
    </div>
  );
};

export default StepEditorHttpConfiguration;
