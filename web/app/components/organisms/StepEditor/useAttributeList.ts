import { useCallback, useRef, useState } from "react";
import { AttributeType, Step } from "@/app/api";
import {
  Attribute,
  AttributeRoleType,
  buildAttributesFromStep,
} from "./stepEditorUtils";
import { validateDefaultValue } from "@/utils/stepUtils";

interface FieldUpdate {
  field: keyof Attribute;
  value: any;
}

const applyAttributeFieldSideEffects = (
  updated: Attribute,
  update: FieldUpdate,
  t: (key: string) => string
): Attribute => {
  const { field, value } = update;
  if (
    (field === "defaultValue" || field === "dataType") &&
    (updated.role === "optional" || updated.role === "const") &&
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

  if (field === "role") {
    if (value !== "optional" && value !== "const") {
      updated.validationError = undefined;
    }
    if (value !== "required") {
      updated.matchLanguage = undefined;
      updated.matchScript = undefined;
    }
    if (value === "const") {
      updated.collect = "first";
    }
    if (value === "output" || value === "meta") {
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
        role: "required" as AttributeRoleType,
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
            { field, value },
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

  const resetAttributes = useCallback((nextStep: Step | null) => {
    setAttributes(buildAttributesFromStep(nextStep));
  }, []);

  return {
    attributes,
    addAttribute,
    updateAttribute,
    removeAttribute,
    resetAttributes,
  };
}
