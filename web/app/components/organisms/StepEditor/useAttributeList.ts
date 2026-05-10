import { useCallback, useRef, useState } from "react";
import { AttributeType, InputCollect, Step } from "@/app/api";
import {
  Attribute,
  AttributeRoleType,
  buildAttributesFromStep,
} from "./stepEditorUtils";
import { validateDefaultValue } from "@/utils/stepUtils";
import { INPUT_COLLECT_TYPES } from "./stepEditorConstants";

const applyAttributeFieldSideEffects = (
  updated: Attribute,
  field: keyof Attribute,
  value: any,
  t: (key: string) => string
): Attribute => {
  if (
    (field === "defaultValue" || field === "dataType") &&
    (updated.attrType === "optional" || updated.attrType === "const") &&
    updated.defaultValue
  ) {
    const validation = validateDefaultValue(
      updated.defaultValue,
      updated.dataType
    );
    updated.validationError = validation.valid
      ? undefined
      : t(validation.errorKey ?? "");
  }

  if (field === "attrType") {
    if (value !== "optional" && value !== "const") {
      updated.validationError = undefined;
    }
    if (value !== "input") {
      updated.matchLanguage = undefined;
      updated.matchScript = undefined;
    }
    if (value === "const") {
      updated.collect = "first";
    }
    if (value === "output") {
      updated.forEach = false;
      updated.collect = "first";
    }
  }

  return updated;
};

export function useAttributeList(
  step: Step | null,
  t: (key: string) => string
) {
  const [attributes, setAttributes] = useState<Attribute[]>(() =>
    buildAttributesFromStep(step)
  );
  const attributeCounterRef = useRef(0);

  const addAttribute = useCallback(() => {
    setAttributes((current) => [
      ...current,
      {
        id: `attr-${++attributeCounterRef.current}`,
        attrType: "input" as AttributeRoleType,
        name: "",
        dataType: AttributeType.String,
      },
    ]);
  }, []);

  const updateAttribute = useCallback(
    (id: string, field: keyof Attribute, value: any) => {
      setAttributes((current) =>
        current.map((attr) => {
          if (attr.id !== id) return attr;
          return applyAttributeFieldSideEffects(
            { ...attr, [field]: value },
            field,
            value,
            t
          );
        })
      );
    },
    [t]
  );

  const removeAttribute = useCallback((id: string) => {
    setAttributes((current) => current.filter((attr) => attr.id !== id));
  }, []);

  const cycleAttributeType = useCallback(
    (id: string, currentType: AttributeRoleType) => {
      const types: AttributeRoleType[] = [
        "input",
        "optional",
        "const",
        "output",
      ];
      const currentIndex = types.indexOf(currentType);
      const nextIndex = (currentIndex + 1) % types.length;
      updateAttribute(id, "attrType", types[nextIndex]);
    },
    [updateAttribute]
  );

  const cycleInputCollect = useCallback(
    (id: string, currentCollect: InputCollect = "first") => {
      const currentIndex = INPUT_COLLECT_TYPES.indexOf(currentCollect);
      const nextIndex =
        currentIndex >= 0 ? (currentIndex + 1) % INPUT_COLLECT_TYPES.length : 0;
      updateAttribute(id, "collect", INPUT_COLLECT_TYPES[nextIndex]);
    },
    [updateAttribute]
  );

  const resetAttributes = useCallback((nextStep: Step | null) => {
    setAttributes(buildAttributesFromStep(nextStep));
  }, []);

  return {
    attributes,
    addAttribute,
    updateAttribute,
    removeAttribute,
    cycleAttributeType,
    cycleInputCollect,
    resetAttributes,
  };
}
