import { useMemo } from "react";
import { Step, ExecutionResult } from "@/app/api";
import { getProgressIcon } from "@/utils/progressUtils";
import { useStepProgress } from "@/app/hooks/useStepProgress";
import { useT } from "@/app/i18n";
import { useLocale } from "@/app/store/i18nStore";
import { stepDisplayInfo } from "@/app/components/molecules/StepShared/stepDisplayInfo";
import { buildTooltipSections } from "./footerTooltipSections";

export const useFooter = (
  step: Step,
  flowId?: string,
  execution?: ExecutionResult
) => {
  const t = useT();
  const locale = useLocale();
  const progressState = useStepProgress(step.id, flowId, execution);

  const displayInfo = useMemo(() => stepDisplayInfo(step), [step]);
  const tooltipSections = useMemo(
    () =>
      buildTooltipSections(
        step,
        { execution, flowId, locale, progressState },
        t
      ),
    [execution, flowId, locale, progressState, step, t]
  );

  const hasFlowProgress = !!flowId && progressState.flowId === flowId;
  const ProgressIcon = getProgressIcon(progressState.status);

  return {
    displayInfo,
    tooltipSections,
    hasFlowProgress,
    ProgressIcon,
    progressStatus: progressState.status,
  };
};
