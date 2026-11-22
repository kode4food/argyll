import React, { useRef } from "react";
import { useUpdateNodeInternals } from "@xyflow/react";
import { Step } from "@/app/api";
import { useHandlePositions } from "@/app/hooks/useHandlePositions";

export const useNodeHandles = (id: string, step: Step) => {
  const widgetRef = useRef<HTMLDivElement | null>(null);
  const updateNodeInternals = useUpdateNodeInternals();

  const { allHandles, summaryInputTop, summaryOutputTop } = useHandlePositions(
    step,
    widgetRef
  );

  React.useEffect(() => {
    updateNodeInternals(id);
  }, [allHandles, id, updateNodeInternals]);

  return { widgetRef, allHandles, summaryInputTop, summaryOutputTop };
};
