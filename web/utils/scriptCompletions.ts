import { SCRIPT_LANGUAGE_ARGYLL, type Step } from "@/app/api";

interface AttributeCompletion {
  name: string;
  role: string;
  mappingName?: string;
}

// attribute names come from Step authors, so only the engine's own field
// names are safe as bare dot segments
const ARGYLL_DOT_NAME = /^[A-Za-z_][A-Za-z0-9_]*$/;

const path = (language: string, parts: string[]): string => {
  if (language !== SCRIPT_LANGUAGE_ARGYLL) {
    return `value${parts.map((part) => `[${JSON.stringify(part)}]`).join("")}`;
  }
  return `value${parts
    .map((part) =>
      ARGYLL_DOT_NAME.test(part) ? `.${part}` : `[${JSON.stringify(part)}]`
    )
    .join("")}`;
};

export const stepAttributeCompletions = (
  attributes: AttributeCompletion[],
  includeOutputs: boolean
): string[] => [
  ...new Set(
    attributes
      .filter((attr) => includeOutputs || attr.role !== "output")
      .map((attr) => attr.mappingName?.trim() || attr.name.trim())
      .filter(Boolean)
  ),
];

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
  [...attributes].sort().forEach((name) => {
    paths.push(
      ["attributes", name],
      ["attributes", name, "role"],
      ["attributes", name, "type"],
      ["attributes", name, "compensated"]
    );
  });
  const tagTest = (tag: string): string =>
    language === SCRIPT_LANGUAGE_ARGYLL
      ? `${JSON.stringify(tag)} in value.tags`
      : `has(value["tags"], ${JSON.stringify(tag)})`;

  return [
    ...paths.map((parts) => path(language, parts)),
    ...[...tags].sort().map(tagTest),
  ];
};
