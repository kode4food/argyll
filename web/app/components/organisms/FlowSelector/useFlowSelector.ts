import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useT } from "@/app/i18n";
import { useKeyboardShortcuts } from "@/app/hooks/useKeyboardShortcuts";
import {
  useFlows,
  useFlowsHasMore,
  useFlowsLoading,
  useLoadFlows,
  useLoadMoreFlows,
  useSelectedFlow,
  useSetVisibleFlowIDs,
} from "@/app/store/flowStore";
import { useFlowDropdownManagement } from "./useFlowDropdownManagement";
import { useFlowFromUrl } from "./useFlowFromUrl";

const VISIBLE_FLOW_IDS_DELAY_MS = 150;
const VISIBLE_FLOW_ID_SELECTOR = "[data-flow-id]";
const SEARCH_FOCUS_DELAY_MS = 100;

const visibleFlowIDs = (menu: HTMLDivElement): string[] => {
  const top = menu.scrollTop;
  const bottom = top + menu.clientHeight;
  const items = Array.from(
    menu.querySelectorAll<HTMLElement>(VISIBLE_FLOW_ID_SELECTOR)
  );

  return items
    .filter((item) => {
      if (item.offsetHeight === 0) {
        return true;
      }

      const itemTop = item.offsetTop;
      const itemBottom = itemTop + item.offsetHeight;
      return itemBottom > top && itemTop < bottom;
    })
    .map((item) => item.dataset.flowId)
    .filter((flowID): flowID is string => !!flowID);
};

export const useFlowSelector = () => {
  const t = useT();
  const navigate = useNavigate();
  useFlowFromUrl();
  const flows = useFlows();
  const selectedFlow = useSelectedFlow();
  const flowsHasMore = useFlowsHasMore();
  const flowsLoading = useFlowsLoading();
  const loadFlows = useLoadFlows();
  const loadMoreFlows = useLoadMoreFlows();
  const setVisibleFlowIDs = useSetVisibleFlowIDs();

  const dropdown = useFlowDropdownManagement(flows, selectedFlow);
  const {
    showDropdown,
    setShowDropdown,
    scrollTop,
    searchInputRef,
    dropdownRef,
    filteredFlows,
  } = dropdown;

  const [showShortcutsModal, setShowShortcutsModal] = useState(false);

  React.useEffect(() => {
    if (!showDropdown) {
      setVisibleFlowIDs([]);
      return () => {
        setVisibleFlowIDs([]);
      };
    }

    const timeout = setTimeout(() => {
      const menu = dropdownRef.current;
      if (!menu) {
        setVisibleFlowIDs([]);
        return;
      }

      setVisibleFlowIDs(visibleFlowIDs(menu));
    }, VISIBLE_FLOW_IDS_DELAY_MS);

    return () => {
      clearTimeout(timeout);
    };
  }, [dropdownRef, filteredFlows, scrollTop, setVisibleFlowIDs, showDropdown]);

  React.useEffect(() => {
    if (!showDropdown) {
      return;
    }
    void loadFlows();
  }, [loadFlows, showDropdown]);

  useKeyboardShortcuts(
    [
      {
        key: "/",
        description: t("flowSelector.focusSearch"),
        handler: () => {
          if (!showDropdown) {
            setShowDropdown(true);
            setTimeout(
              () => searchInputRef.current?.focus(),
              SEARCH_FOCUS_DELAY_MS
            );
          }
        },
      },
      {
        key: "?",
        description: t("flowSelector.showShortcuts"),
        handler: () => {
          setShowShortcutsModal(true);
        },
      },
    ],
    !showShortcutsModal
  );

  return {
    dropdown,
    selectedFlow,
    flowsHasMore,
    flowsLoading,
    loadMoreFlows,
    showShortcutsModal,
    closeShortcutsModal: () => setShowShortcutsModal(false),
    navigateToOverview: () => navigate("/"),
  };
};
