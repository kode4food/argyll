require("@testing-library/jest-dom");

const { TextDecoder, TextEncoder } = require("node:util");
const { deserialize, serialize } = require("node:v8");

global.TextDecoder = TextDecoder;
global.TextEncoder = TextEncoder;
global.structuredClone = (value) => deserialize(serialize(value));

class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}

global.ResizeObserver = ResizeObserverMock;

class DOMMatrixReadOnlyMock {
  constructor() {
    this.m11 = 1;
    this.m22 = 1;
    this.m41 = 0;
    this.m42 = 0;
  }
}

global.DOMMatrixReadOnly = DOMMatrixReadOnlyMock;

const baseMessages = require("./app/i18n/en-US.json");
const commonMessages = require("./app/i18n/common.json");
const messages = { ...commonMessages, ...baseMessages };

const {
  interpolate,
  isPluralForms,
  selectPluralForm,
} = require("./app/i18n/i18nUtils");

const t = (key, vars) => {
  const message = messages[key];
  if (!message) return key;

  if (isPluralForms(message)) {
    const count = vars?.count;
    if (typeof count !== "number") {
      console.warn(
        `Plural message "${key}" requires a numeric 'count' variable`
      );
      return key;
    }
    return interpolate(selectPluralForm(message, count), vars);
  }

  return interpolate(message, vars);
};

jest.mock("@/app/i18n", () => ({
  I18nProvider: ({ children }) => children,
  useT: () => t,
}));

const durationUnits = {
  d: { singular: "day", plural: "days", ms: 24 * 60 * 60 * 1000 },
  day: { singular: "day", plural: "days", ms: 24 * 60 * 60 * 1000 },
  days: { singular: "day", plural: "days", ms: 24 * 60 * 60 * 1000 },
  h: { singular: "hour", plural: "hours", ms: 60 * 60 * 1000 },
  hour: { singular: "hour", plural: "hours", ms: 60 * 60 * 1000 },
  hours: { singular: "hour", plural: "hours", ms: 60 * 60 * 1000 },
  heure: { singular: "heure", plural: "heures", ms: 60 * 60 * 1000 },
  heures: { singular: "heure", plural: "heures", ms: 60 * 60 * 1000 },
  m: { singular: "minute", plural: "minutes", ms: 60 * 1000 },
  minute: { singular: "minute", plural: "minutes", ms: 60 * 1000 },
  minutes: { singular: "minute", plural: "minutes", ms: 60 * 1000 },
  s: { singular: "second", plural: "seconds", ms: 1000 },
  second: { singular: "second", plural: "seconds", ms: 1000 },
  seconds: { singular: "second", plural: "seconds", ms: 1000 },
};

const formatDuration = (value) => {
  const units = [
    durationUnits.d,
    durationUnits.h,
    durationUnits.m,
    durationUnits.s,
  ];
  const unit = units.find((candidate) => value >= candidate.ms);
  if (!unit) return `${value} milliseconds`;
  const amount = Math.floor(value / unit.ms);
  const label = amount === 1 ? unit.singular : unit.plural;
  return `${amount} ${label}`;
};

const parseDuration = (value) => {
  const match = value
    .trim()
    .toLowerCase()
    .match(/^(\d+(?:[\.,]\d+)?)\s*([a-z]+)$/);
  if (!match) throw new Error("invalid duration");
  const amount = Number(match[1].replace(",", "."));
  const unit = durationUnits[match[2]];
  if (!unit) throw new Error("invalid duration unit");
  return amount * unit.ms;
};

jest.mock("enhanced-ms/locales/de", () => ({
  __esModule: true,
  default: { language: "de" },
}));

jest.mock("enhanced-ms/locales/en", () => ({
  __esModule: true,
  default: { language: "en" },
}));

jest.mock("enhanced-ms/locales/fr", () => ({
  __esModule: true,
  default: { language: "fr" },
}));

jest.mock("enhanced-ms/locales/it", () => ({
  __esModule: true,
  default: { language: "it" },
}));

jest.mock("enhanced-ms", () => ({
  compileLanguage: (definition) => ({
    decimalSeparator: definition.language === "fr" ? "," : ".",
    matcherRegex:
      definition.language === "fr"
        ? /d|h|heure|heures|m|minute|minutes|s|second|seconds/g
        : /d|day|days|h|hour|hours|m|minute|minutes|s|second|seconds/g,
  }),
  createMs: () => (value) =>
    typeof value === "number" ? formatDuration(value) : parseDuration(value),
}));
