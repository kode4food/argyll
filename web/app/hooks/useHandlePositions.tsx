import React, { useState, useCallback, useEffect } from "react";
import { Step } from "@/app/api";
import {
  groupAttributesByRole,
  generateHandleId,
  HandlePositions,
  HandlePosition,
} from "@/utils/stepNodeUtils";

export const useHandlePositions = (
  step: Step,
  stepWidgetRef: React.RefObject<HTMLDivElement | null>
): {
  handlePositions: HandlePositions;
  allHandles: HandlePosition[];
  summaryInputTop: number | undefined;
  summaryOutputTop: number | undefined;
} => {
  const [handlePositions, setHandlePositions] = useState<HandlePositions>({
    required: [],
    optional: [],
    output: [],
  });

  const updateHandlePositions = useCallback(() => {
    if (!stepWidgetRef.current) return;

    const { required, optional, output } = groupAttributesByRole(
      step.attributes || {}
    );

    interface HandleDescriptor {
      type: string;
      name: string;
      handleType: "input" | "output";
    }

    const getHandlePosition = (
      element: Element,
      desc: HandleDescriptor
    ): HandlePosition => {
      const relativeTop =
        (element as HTMLElement).offsetTop +
        (element as HTMLElement).offsetHeight / 2;
      return {
        id: generateHandleId(desc.type, desc.name),
        top: relativeTop,
        argName: desc.name,
        handleType: desc.handleType,
      };
    };

    const findHandles = (
      argType: string,
      argNames: string[],
      handleType: "input" | "output"
    ): HandlePosition[] => {
      return argNames
        .map((name) => {
          const element = stepWidgetRef.current?.querySelector(
            `[data-arg-type="${argType}"][data-arg-name="${name}"]`
          );
          return element
            ? getHandlePosition(element, { type: argType, name, handleType })
            : null;
        })
        .filter((handle): handle is HandlePosition => handle !== null);
    };

    setHandlePositions({
      required: findHandles("required", required, "input"),
      optional: findHandles("optional", optional, "input"),
      output: findHandles("output", output, "output"),
    });
  }, [step.attributes]);

  useEffect(() => {
    updateHandlePositions();
  }, [updateHandlePositions]);

  const allHandles: HandlePosition[] = [
    ...handlePositions.required,
    ...handlePositions.optional,
    ...handlePositions.output,
  ];

  // So the collapsed summary handle lands near the real handles instead of
  // snapping to the node's vertical center, which keeps edges from jumping
  // when detail collapses.
  const midpointTop = (handles: HandlePosition[]): number | undefined => {
    if (handles.length === 0) return undefined;
    const tops = handles.map((handle) => handle.top);
    return (Math.min(...tops) + Math.max(...tops)) / 2;
  };

  const summaryInputTop = midpointTop([
    ...handlePositions.required,
    ...handlePositions.optional,
  ]);
  const summaryOutputTop = midpointTop(handlePositions.output);

  return {
    handlePositions,
    allHandles,
    summaryInputTop,
    summaryOutputTop,
  };
};
