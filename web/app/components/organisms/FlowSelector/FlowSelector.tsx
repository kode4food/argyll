import React, { lazy, Suspense } from "react";
import { createPortal } from "react-dom";
import { IconNavigateOverview } from "@/utils/iconRegistry";
import { useT } from "@/app/i18n";
import SegmentedGroup from "@/app/components/molecules/SegmentedGroup";
import FlowSelectorDropdown from "./FlowSelectorDropdown";
import { useFlowSelector } from "./useFlowSelector";
import styles from "./FlowSelector.module.css";

const KeyboardShortcutsModal = lazy(
  () => import("@/app/components/molecules/KeyboardShortcutsModal")
);

const FlowSelectorContent: React.FC = () => {
  const t = useT();
  const {
    dropdown,
    selectedFlow,
    flowsHasMore,
    flowsLoading,
    loadMoreFlows,
    showShortcutsModal,
    closeShortcutsModal,
    navigateToOverview,
  } = useFlowSelector();

  return (
    <div className={styles.selector}>
      <h1 className={styles.titleText}>{t("flowSelector.title")}</h1>
      <a
        href="https://www.argyll.app/"
        target="_blank"
        rel="noreferrer"
        className={styles.titleLink}
        aria-label={t("flowSelector.siteLabel")}
      >
        <img
          src="/argyll-logo.png"
          alt={t("flowSelector.logoAlt")}
          className={styles.icon}
          width={123}
          height={77}
        />
      </a>
      <SegmentedGroup className={styles.selectorGroup}>
        <FlowSelectorDropdown
          {...dropdown}
          flowsHasMore={flowsHasMore}
          flowsLoading={flowsLoading}
          loadMoreFlows={loadMoreFlows}
        />
        {selectedFlow && (
          <button
            onClick={navigateToOverview}
            className={styles.navButtonSegment}
            title={t("flowSelector.backToOverview")}
            aria-label={t("flowSelector.backToOverview")}
          >
            <IconNavigateOverview
              className={styles.buttonIcon}
              aria-hidden="true"
            />
          </button>
        )}
      </SegmentedGroup>
      <Suspense fallback={null}>
        {showShortcutsModal &&
          createPortal(
            <KeyboardShortcutsModal
              isOpen={showShortcutsModal}
              onClose={closeShortcutsModal}
            />,
            document.body
          )}
      </Suspense>
    </div>
  );
};

const FlowSelector: React.FC = () => <FlowSelectorContent />;

export default FlowSelector;
