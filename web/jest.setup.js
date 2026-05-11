require("@testing-library/jest-dom");

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
