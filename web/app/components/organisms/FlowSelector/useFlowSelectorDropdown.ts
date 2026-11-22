import React from "react";
import { FlowSummary } from "@/app/api";
import { useAnchoredList } from "@/app/hooks/useAnchoredList";
import { useFlowCompensating } from "@/app/store/flowStore";
import dropdownStyles from "@/app/styles/components/dropdown.module.css";
import { getProgressIcon } from "@/utils/progressUtils";
import { mapFlowStatusToProgressStatus } from "./flowSelectorUtils";
import styles from "./FlowSelector.module.css";

const LOAD_MORE_THRESHOLD_PX = 40;

export interface FlowSelectorDropdownOptions {
  showDropdown: boolean;
  selectedIndex: number;
  selectedFlow: string | null;
  filteredFlows: FlowSummary[];
  flows: FlowSummary[];
  setScrollTop: (scrollTop: number) => void;
  flowsHasMore: boolean;
  flowsLoading: boolean;
  loadMoreFlows: () => Promise<void>;
}

export const useFlowSelectorDropdown = ({
  showDropdown,
  selectedIndex,
  selectedFlow,
  filteredFlows,
  flows,
  setScrollTop,
  flowsHasMore,
  flowsLoading,
  loadMoreFlows,
}: FlowSelectorDropdownOptions) => {
  const isCompensating = useFlowCompensating();

  const handleDropdownScroll = (e: React.UIEvent<HTMLDivElement>) => {
    setScrollTop(e.currentTarget.scrollTop);

    if (!flowsHasMore || flowsLoading) {
      return;
    }
    const target = e.currentTarget;
    const isNearBottom =
      target.scrollTop + target.clientHeight >=
      target.scrollHeight - LOAD_MORE_THRESHOLD_PX;
    if (isNearBottom) {
      void loadMoreFlows();
    }
  };

  const triggerRef = React.useRef<HTMLDivElement>(null);
  const menuRef = React.useRef<HTMLDivElement>(null);
  useAnchoredList({
    isOpen: showDropdown,
    anchorRef: triggerRef,
    listRef: menuRef,
    shouldMatchWidth: true,
  });

  const flowRows = filteredFlows.map((flow, index) => {
    const isHighlighted = selectedIndex === index;
    const isSelected = selectedFlow === flow.id;
    const progressStatus =
      isSelected && isCompensating
        ? "compensating"
        : mapFlowStatusToProgressStatus(flow.status);
    const className = [
      dropdownStyles.item,
      styles.dropdownItem,
      isHighlighted && styles.dropdownItemHighlighted,
      isSelected && dropdownStyles.itemActive,
      isSelected && styles.dropdownItemSelected,
    ]
      .filter(Boolean)
      .join(" ");
    return {
      flow,
      index,
      className,
      progressStatus,
      StatusIcon: getProgressIcon(progressStatus),
    };
  });

  const selectedSummary = flows.find((w) => w.id === selectedFlow);
  const selectedProgressStatus = isCompensating
    ? "compensating"
    : mapFlowStatusToProgressStatus(selectedSummary?.status ?? "pending");
  const SelectedStatusIcon = getProgressIcon(selectedProgressStatus);

  return {
    handleDropdownScroll,
    triggerRef,
    menuRef,
    flowRows,
    selectedProgressStatus,
    SelectedStatusIcon,
  };
};
