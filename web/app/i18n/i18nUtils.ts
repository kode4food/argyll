export type PluralForms = {
  zero?: string;
  one?: string;
  other: string;
};

export type MessageValue = string | PluralForms;
export type Vars = Record<string, string | number>;

export const interpolate = (template: string, vars?: Vars): string => {
  if (!vars) return template;
  return template.replace(/\{(\w+)\}/g, (_, key: string) => {
    if (Object.prototype.hasOwnProperty.call(vars, key)) {
      return String(vars[key]);
    }
    return `{${key}}`;
  });
};

export const isPluralForms = (value: MessageValue): value is PluralForms => {
  return (
    typeof value === "object" &&
    value !== null &&
    "other" in value &&
    typeof value.other === "string"
  );
};

export const selectPluralForm = (forms: PluralForms, count: number): string => {
  if (count === 0 && forms.zero !== undefined) return forms.zero;
  if (count === 1 && forms.one !== undefined) return forms.one;
  return forms.other;
};
