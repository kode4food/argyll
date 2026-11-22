import React from "react";
import { Step } from "@/app/api";
import { useSpaces, useSpaceSelection } from "@/app/store/flowStore";
import { type LucideIcon } from "@/utils/iconRegistry";

export const useSpaceOptions = (emptyLabel: string, Icon?: LucideIcon) => {
  const spaces = useSpaces();
  return React.useMemo(
    () => [
      { value: "", label: emptyLabel, Icon },
      ...spaces.map((s) => ({
        value: s.id,
        label: s.name,
        title: s.description,
        Icon,
      })),
    ],
    [spaces, emptyLabel, Icon]
  );
};

// The engine plans within the Space, so a goal outside it could never run
export const useSpaceScopedSteps = (steps: Step[], spaceId: string | null) => {
  const spaceSelection = useSpaceSelection();
  return React.useMemo(() => {
    const selected = spaceId ? spaceSelection[spaceId] : null;
    return selected ? steps.filter((s) => selected.has(s.id)) : steps;
  }, [steps, spaceId, spaceSelection]);
};
