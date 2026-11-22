import React from "react";
import { stepAttributeCompletions } from "@/utils/scriptCompletions";
import type { Attribute } from "./stepEditorUtils";

export const useAttributeCompletions = (attributes: Attribute[]) => {
  const predicateCompletions = React.useMemo(
    () => stepAttributeCompletions(attributes, false),
    [attributes]
  );
  const scriptCompletions = React.useMemo(
    () => stepAttributeCompletions(attributes, true),
    [attributes]
  );
  return { predicateCompletions, scriptCompletions };
};
