require("@testing-library/jest-dom");

class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}

global.ResizeObserver = ResizeObserverMock;

const baseMessages = require("./app/i18n/en-US.json");
const commonMessages = require("./app/i18n/common.json");
const messages = { ...commonMessages, ...baseMessages };

const interpolate = (template, vars) => {
  if (!vars) {
    return template;
  }
  return template.replace(/\{(\w+)\}/g, (_, key) => {
    if (Object.prototype.hasOwnProperty.call(vars, key)) {
      return String(vars[key]);
    }
    return `{${key}}`;
  });
};

const isPluralForms = (value) => {
  return (
    typeof value === "object" &&
    value !== null &&
    "other" in value &&
    typeof value.other === "string"
  );
};

const selectPluralForm = (forms, count) => {
  if (count === 0 && forms.zero !== undefined) {
    return forms.zero;
  }
  if (count === 1 && forms.one !== undefined) {
    return forms.one;
  }
  return forms.other;
};

const t = (key, vars) => {
  const message = messages[key];

  // If message not found, return the key
  if (!message) {
    return key;
  }

  // Handle plural forms
  if (isPluralForms(message)) {
    const count = vars?.count;
    if (typeof count !== "number") {
      console.warn(
        `Plural message "${key}" requires a numeric 'count' variable`
      );
      return key;
    }
    const template = selectPluralForm(message, count);
    return interpolate(template, vars);
  }

  // Handle simple string messages
  return interpolate(message, vars);
};

jest.mock("@/app/i18n", () => ({
  I18nProvider: ({ children }) => children,
  useT: () => t,
}));
