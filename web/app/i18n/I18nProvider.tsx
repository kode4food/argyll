import React, { createContext, useCallback, useContext, useMemo } from "react";
import commonMessages from "./common.json";
import defaultMessages from "./en-US.json";

type PluralForms = {
  zero?: string;
  one?: string;
  other: string;
};

type MessageValue = string | PluralForms;
type Messages = Record<string, MessageValue>;
type Vars = Record<string, string | number>;

interface I18nContextValue {
  t: (key: string, vars?: Vars) => string;
  locale: string;
  messages: Messages;
}

const I18nContext = createContext<I18nContextValue | null>(null);

interface I18nProviderProps {
  children: React.ReactNode;
  locale?: string;
  messages?: Messages;
}

const interpolate = (template: string, vars?: Vars) => {
  if (!vars) {
    return template;
  }
  return template.replace(/\{(\w+)\}/g, (_, key: string) => {
    if (Object.prototype.hasOwnProperty.call(vars, key)) {
      return String(vars[key]);
    }
    return `{${key}}`;
  });
};

const isPluralForms = (value: MessageValue): value is PluralForms => {
  return (
    typeof value === "object" &&
    value !== null &&
    "other" in value &&
    typeof value.other === "string"
  );
};

const selectPluralForm = (forms: PluralForms, count: number): string => {
  if (count === 0 && forms.zero !== undefined) {
    return forms.zero;
  }
  if (count === 1 && forms.one !== undefined) {
    return forms.one;
  }
  return forms.other;
};

const I18nProvider: React.FC<I18nProviderProps> = ({
  children,
  locale = "en-US",
  messages = defaultMessages,
}) => {
  const mergedMessages = useMemo<Messages>(
    () => ({ ...commonMessages, ...messages }),
    [messages]
  );

  const t = useCallback(
    (key: string, vars?: Vars) => {
      const message = mergedMessages[key];

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
    },
    [mergedMessages]
  );

  const value = useMemo(
    () => ({
      t,
      locale,
      messages: mergedMessages,
    }),
    [t, locale, mergedMessages]
  );

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
};

const useT = () => {
  const ctx = useContext(I18nContext);
  if (!ctx) {
    throw new Error("useT must be used within an I18nProvider");
  }
  return ctx.t;
};

export { I18nProvider, useT };
