import React, { useState, useCallback, useEffect } from "react";
import { Step } from "@/app/api";
import {
  groupAttributesByRole,
  generateHandleId,
  HandlePositions,
  HandlePosition,
} from "@/utils/stepNodeUtils";

/**
 * Hook that manages handle positions for a step node
 * Calculates DOM-based positions for connection handles
 *
 * @param step - The step data
 * @param stepWidgetRef - Reference to the step widget container
 * @returns Object containing handle positions and all handles array
 */
export const useHandlePositions = (
  step: Step,
  stepWidgetRef: React.RefObject<HTMLDivElement | null>
): { handlePositions: HandlePositions; allHandles: HandlePosition[] } => {
  const [handlePositions, setHandlePositions] = useState<HandlePositions>({
    required: [],
    optional: [],
    output: [],
  });

  /**
   * Updates handle positions by querying the DOM
   * Finds elements with matching data attributes and calculates their positions
   */
  const updateHandlePositions = useCallback(() => {
    if (!stepWidgetRef.current) return;

    const { required, optional, output } = groupAttributesByRole(
      step.attributes || {}
    );

    /**
     * Gets the position for a single handle element
     */
    const getHandlePosition = (
      element: Element,
      type: string,
      name: string,
      handleType: "input" | "output"
    ): HandlePosition => {
      const relativeTop =
        (element as HTMLElement).offsetTop +
        (element as HTMLElement).offsetHeight / 2;
      return {
        id: generateHandleId(type, name),
        top: relativeTop,
        argName: name,
        handleType,
      };
    };

    /**
     * Finds all handle positions for a given attribute type
     */
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
            ? getHandlePosition(element, argType, name, handleType)
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

  /**
   * Run updateHandlePositions whenever the step attributes change
   */
  useEffect(() => {
    updateHandlePositions();
  }, [updateHandlePositions]);

  /**
   * Flatten all handles into a single array for easy iteration
   */
  const allHandles: HandlePosition[] = [
    ...handlePositions.required,
    ...handlePositions.optional,
    ...handlePositions.output,
  ];

  return {
    handlePositions,
    allHandles,
  };
};
