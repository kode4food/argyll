import { compileLanguage, createMs } from "enhanced-ms";

export type ParseResult = { valid: true; ms: number } | { valid: false };

export const isNumericOnly = (
  value: string,
  decimalSeparator: "." | ","
): boolean => {
  const escaped = decimalSeparator === "," ? "," : "\\.";
  const regex = new RegExp(`^\\d+(?:[${escaped}]\\d+)?$`);
  return regex.test(value);
};

const tryParseNumber = (value: string, decimalSeparator: "." | ","): number => {
  const normalized = decimalSeparator === "," ? value.replace(",", ".") : value;
  return Number(normalized);
};

const hasUnitToken = (value: string, matcherRegex: RegExp): boolean => {
  if (matcherRegex.global) matcherRegex.lastIndex = 0;
  const matches = value.match(matcherRegex) ?? [];
  return matches.some((match) => !/[0-9]/.test(match));
};

const hasNumberAndUnit = (value: string, matcherRegex: RegExp): boolean => {
  if (!/[0-9]/.test(value)) return false;
  return hasUnitToken(value, matcherRegex);
};

export const parseUserDuration = (
  input: string,
  language: ReturnType<typeof compileLanguage>,
  ms: ReturnType<typeof createMs>
): ParseResult => {
  const trimmed = input.trim();
  if (!trimmed) return { valid: true, ms: 0 };
  if (trimmed.startsWith("-")) return { valid: false };

  if (isNumericOnly(trimmed, language.decimalSeparator)) {
    return {
      valid: true,
      ms: tryParseNumber(trimmed, language.decimalSeparator),
    };
  }

  if (!hasNumberAndUnit(trimmed.toLowerCase(), language.matcherRegex)) {
    return { valid: false };
  }

  try {
    const parsed = ms(trimmed);
    if (parsed === null) return { valid: false };
    return { valid: true, ms: parsed };
  } catch {
    return { valid: false };
  }
};
