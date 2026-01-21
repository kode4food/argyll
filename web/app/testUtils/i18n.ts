import commonMessages from "@/app/i18n/common.json";
import enUS from "@/app/i18n/en-US.json";

type Messages = typeof commonMessages & typeof enUS;
type MessageValue = Messages[keyof Messages];

const interpolate = (
  template: string,
  vars: Record<string, string | number> = {}
): string => {
  return template.replace(/\{(\w+)\}/g, (_, key) =>
    Object.prototype.hasOwnProperty.call(vars, key) ? String(vars[key]) : ""
  );
};

const mergedMessages: Record<string, MessageValue> = {
  ...commonMessages,
  ...enUS,
};

export const t = (
  key: keyof Messages | string,
  vars?: Record<string, string | number>
): string => {
  const value = mergedMessages[key as string];
  if (!value) {
    return String(key);
  }
  if (typeof value === "string") {
    return interpolate(value, vars);
  }
  const count = vars?.count;
  if (typeof count !== "number") {
    return String(key);
  }
  const form =
    count === 0 && value.zero !== undefined
      ? "zero"
      : count === 1 && value.one !== undefined
        ? "one"
        : "other";
  return interpolate(value[form], vars);
};

export const tPlural = (
  key: keyof Messages,
  count: number,
  vars: Record<string, string | number> = {}
): string => {
  const value = mergedMessages[key as string] as MessageValue;
  if (!value || typeof value !== "object") {
    throw new Error(`i18n key "${String(key)}" is not a plural object`);
  }
  const form =
    count === 0 && "zero" in value
      ? "zero"
      : count === 1 && "one" in value
        ? "one"
        : "other";
  return interpolate(value[form], { ...vars, count });
};

export { enUS, commonMessages };
