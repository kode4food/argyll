import { SCRIPT_LANGUAGE_JPATH, type Step } from "@/app/api";

interface AttributeCompletion {
  name: string;
  role: string;
  mappingName?: string;
}

const path = (language: string, parts: string[]): string =>
  `${language === SCRIPT_LANGUAGE_JPATH ? "$" : "value"}${parts
    .map((part) => `[${JSON.stringify(part)}]`)
    .join("")}`;

export const stepAttributeCompletions = (
  attributes: AttributeCompletion[],
  language: string,
  includeOutputs: boolean
): string[] =>
  [
    ...new Set(
      attributes
        .filter((attr) => includeOutputs || attr.role !== "output")
        .map((attr) => attr.mappingName?.trim() || attr.name.trim())
        .filter(Boolean)
    ),
  ].map((name) =>
    language === SCRIPT_LANGUAGE_JPATH ? path(language, [name]) : name
  );

export const spaceSelectorCompletions = (
  steps: Pick<Step, "attributes" | "tags">[],
  language: string
): string[] => {
  const tags = new Set<string>();
  const attributes = new Set<string>();
  steps.forEach((step) => {
    step.tags?.forEach((tag) => tags.add(tag));
    Object.keys(step.attributes).forEach((name) => attributes.add(name));
  });

  const paths = [["tags"], ["type"], ["handling"], ["attributes"]];
  [...tags].sort().forEach((tag) => paths.push(["tags", tag]));
  [...attributes].sort().forEach((name) => {
    paths.push(
      ["attributes", name],
      ["attributes", name, "role"],
      ["attributes", name, "type"],
      ["attributes", name, "compensated"]
    );
  });
  return paths.map((parts) => path(language, parts));
};
