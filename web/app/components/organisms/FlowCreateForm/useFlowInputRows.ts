import { useMemo } from "react";
import {
  IconAttributeStatusMissing,
  IconAttributeStatusSatisfied,
  IconAttributeOptional,
  IconAttributeStatusSkipped,
  LucideIcon,
} from "@/utils/iconRegistry";
import { useT } from "@/app/i18n";
import { FlowInputOption } from "@/utils/flowPlanAttributeOptions";
import { FlowInputStatus, getFlowInputStatus } from "./flowFormUtils";
import styles from "./FlowAttributesSection.module.css";

export interface StatusConfig {
  Icon: LucideIcon;
  className: string;
  label: string;
}

export interface FlowInputRow extends StatusConfig {
  option: FlowInputOption;
  value: string;
}

export const useFlowInputRows = (
  flowInputOptions: FlowInputOption[],
  flowInputValues: Record<string, string>,
  flowInputValuesRaw: Record<string, string>
): FlowInputRow[] => {
  const statusConfig = useFlowInputStatusConfig();
  return flowInputOptions.map((option) => {
    const rawValue = flowInputValuesRaw[option.name] || "";
    const status = getFlowInputStatus(option, rawValue);
    return {
      option,
      value: flowInputValues[option.name] || "",
      ...statusConfig[status],
    };
  });
};

const useFlowInputStatusConfig = () => {
  const t = useT();
  return useMemo(
    (): Record<FlowInputStatus, StatusConfig> => ({
      requiredMissing: {
        Icon: IconAttributeStatusMissing,
        className: styles.badgeRequired,
        label: t("flowCreate.badgeRequiredMissing"),
      },
      requiredSatisfied: {
        Icon: IconAttributeStatusSatisfied,
        className: styles.badgeRequired,
        label: t("flowCreate.badgeRequiredSatisfied"),
      },
      optionalMissing: {
        Icon: IconAttributeOptional,
        className: styles.badgeOptional,
        label: t("flowCreate.badgeOptionalMissing"),
      },
      optionalSatisfied: {
        Icon: IconAttributeStatusSatisfied,
        className: styles.badgeOptional,
        label: t("flowCreate.badgeOptionalSatisfied"),
      },
      outputSatisfied: {
        Icon: IconAttributeStatusSatisfied,
        className: styles.badgeOutput,
        label: t("flowCreate.badgeOutputSatisfied"),
      },
      unreachable: {
        Icon: IconAttributeStatusSkipped,
        className: styles.badgeMuted,
        label: t("flowCreate.badgeUnreachable"),
      },
    }),
    [t]
  );
};
