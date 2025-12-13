import React from "react";
import { Step } from "../../api";
import Tooltip from "../atoms/Tooltip";
import TooltipSection from "../atoms/TooltipSection";
import tooltipStyles from "../atoms/TooltipSection.module.css";
import styles from "./StepPredicate.module.css";

interface StepPredicateProps {
  step: Step;
}

const StepPredicate: React.FC<StepPredicateProps> = ({ step }) => {
  if (!step.predicate) {
    return null;
  }

  const scriptPreview = step.predicate.script
    .split("\n")
    .slice(0, 5)
    .join("\n");
  const lineCount = step.predicate.script.split("\n").length;

  return (
    <div className={`${styles.argsSection} step-args-section`}>
      <Tooltip
        trigger={
          <div className={styles.content}>
            <div className={styles.code}>{step.predicate.script}</div>
          </div>
        }
      >
        <TooltipSection title={`Predicate (${step.predicate.language})`}>
          <div className={tooltipStyles.valueCode}>
            {scriptPreview}
            {lineCount > 5 && (
              <div className={tooltipStyles.codeMore}>
                ... ({lineCount - 5} more lines)
              </div>
            )}
          </div>
        </TooltipSection>
      </Tooltip>
    </div>
  );
};

export default StepPredicate;
