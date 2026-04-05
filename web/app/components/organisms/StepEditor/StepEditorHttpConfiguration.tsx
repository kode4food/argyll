import React from "react";
import DurationInput from "@/app/components/molecules/DurationInput";
import { useT } from "@/app/i18n";
import formStyles from "./StepEditorForm.module.css";
import localStyles from "./StepEditorHttpConfiguration.module.css";

interface StepEditorHttpConfigurationProps {
  endpoint: string;
  healthCheck: string;
  httpTimeout: number;
  setEndpoint: (value: string) => void;
  setHealthCheck: (value: string) => void;
  setHttpTimeout: (value: number) => void;
}

const StepEditorHttpConfiguration: React.FC<
  StepEditorHttpConfigurationProps
> = ({
  endpoint,
  healthCheck,
  httpTimeout,
  setEndpoint,
  setHealthCheck,
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
      </div>
    </div>
  );
};

export default StepEditorHttpConfiguration;
