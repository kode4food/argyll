import React from "react";
import DurationInput from "@/app/components/molecules/DurationInput";
import { HTTPMethod } from "@/app/api";
import { useT } from "@/app/i18n";
import SegmentedGroup from "@/app/components/molecules/SegmentedGroup";
import formStyles from "./StepEditorForm.module.css";
import localStyles from "./StepEditorHttpConfiguration.module.css";
import InlineSelectDropdown from "./InlineSelectDropdown";

interface StepEditorHttpConfigurationProps {
  endpoint: string;
  httpMethod: HTTPMethod;
  healthCheck: string;
  compensate: string;
  httpTimeout: number;
  memoizable: boolean;
  setEndpoint: (value: string) => void;
  setHttpMethod: (value: HTTPMethod) => void;
  setHealthCheck: (value: string) => void;
  setCompensate: (value: string) => void;
  setHttpTimeout: (value: number) => void;
}

const StepEditorHttpConfiguration: React.FC<
  StepEditorHttpConfigurationProps
> = ({
  endpoint,
  httpMethod,
  healthCheck,
  compensate,
  httpTimeout,
  memoizable,
  setEndpoint,
  setHttpMethod,
  setHealthCheck,
  setCompensate,
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
                options={[
                  { value: "POST", label: "POST" },
                  { value: "GET", label: "GET" },
                  { value: "PUT", label: "PUT" },
                  { value: "DELETE", label: "DELETE" },
                ]}
                onChange={(v) => setHttpMethod(v as HTTPMethod)}
              />
            </SegmentedGroup>
          </div>
          <div className={`${formStyles.field} ${formStyles.flex1}`}>
            <label className={formStyles.label}>
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
        <div className={formStyles.field}>
          <label className={formStyles.label}>
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
        <div className={formStyles.field}>
          <label className={formStyles.label}>
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
      </div>
    </div>
  );
};

export default StepEditorHttpConfiguration;
