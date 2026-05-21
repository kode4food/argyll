import { validateDefaultValue } from "@/utils/stepUtils";
import { Attribute, ValidationError } from "./stepEditorTypes";

export function validateAttributesList(
  attributes: Attribute[]
): ValidationError | null {
  const names = new Set<string>();
  for (const attr of attributes) {
    if (!attr.name.trim()) {
      return { key: "stepEditor.attributeNameRequired" };
    }
    if (names.has(attr.name)) {
      return {
        key: "stepEditor.duplicateAttributeName",
        vars: { name: attr.name },
      };
    }
    names.add(attr.name);

    if (attr.role === "required" && attr.matchScript?.trim()) {
      const matchLanguage = attr.matchLanguage?.trim();
      if (!matchLanguage) {
        return {
          key: "stepEditor.matchLanguageRequired",
          vars: { name: attr.name },
        };
      }
    }

    if (
      (attr.role === "optional" || attr.role === "const") &&
      attr.defaultValue
    ) {
      const validation = validateDefaultValue(attr.defaultValue, attr.dataType);
      if (!validation.valid) {
        return {
          key: "stepEditor.invalidDefaultValue",
          vars: {
            name: attr.name,
            reason: validation.errorKey ?? "",
          },
        };
      }
    }

    if (attr.role === "const" && !attr.defaultValue?.trim()) {
      return {
        key: "stepEditor.constDefaultRequired",
        vars: { name: attr.name },
      };
    }
  }
  return null;
}

export function validateMappings(
  attributes: Attribute[]
): ValidationError | null {
  const inputMappingNames = new Set<string>();
  const outputMappingNames = new Set<string>();

  for (const attr of attributes) {
    const mappingName = attr.mappingName?.trim() || "";
    const mappingScript = attr.mappingScript?.trim() || "";
    const mappingLanguage = attr.mappingLanguage?.trim() || "";

    if (
      (attr.role === "const" || attr.role === "meta") &&
      (mappingName || mappingScript)
    ) {
      return {
        key: "stepEditor.constMappingNotAllowed",
        vars: { name: attr.name },
      };
    }

    if (!mappingName && !mappingScript) {
      continue;
    }

    if (mappingScript && !mappingLanguage) {
      return {
        key: "stepEditor.mappingLanguageRequired",
        vars: { name: attr.name },
      };
    }

    if (!mappingName) {
      continue;
    }

    const bucket =
      attr.role === "output" ? outputMappingNames : inputMappingNames;
    if (bucket.has(mappingName)) {
      return {
        key: "stepEditor.duplicateMappingName",
        vars: { name: mappingName },
      };
    }
    bucket.add(mappingName);
  }

  return null;
}
