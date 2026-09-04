import { SCRIPT_LANGUAGE_JPATH, type Step } from "@/app/api";

interface AttributeCompletion {
  name: string;
  role: string;
  mappingName?: string;
}

// attribute names come from Step authors, so only the engine's own field
// names are safe as bare dot segments
const JPATH_DOT_NAME = /^[A-Za-z_][A-Za-z0-9_]*$/;

const path = (language: string, parts: string[]): string => {
  if (language !== SCRIPT_LANGUAGE_JPATH) {
    return `value${parts.map((part) => `[${JSON.stringify(part)}]`).join("")}`;
  }
  return `$${parts
    .map((part) =>
      JPATH_DOT_NAME.test(part) ? `.${part}` : `[${JSON.stringify(part)}]`
    )
    .join("")}`;
};

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
  [...attributes].sort().forEach((name) => {
    paths.push(
      ["attributes", name],
      ["attributes", name, "role"],
      ["attributes", name, "type"],
      ["attributes", name, "compensated"]
    );
  });
  // a tag test is a filter in JPath and a prelude helper in Lua, so neither
  // is a plain path
  const tagTest = (tag: string): string =>
    language === SCRIPT_LANGUAGE_JPATH
      ? `$.tags[?@==${JSON.stringify(tag)}]`
      : `has(value["tags"], ${JSON.stringify(tag)})`;

  return [
    ...paths.map((parts) => path(language, parts)),
    ...[...tags].sort().map(tagTest),
  ];
};
