import React, { createContext, useCallback, useContext, useMemo } from "react";
import commonMessages from "./common.json";
import defaultMessages from "./en-US.json";

type Messages = Record<string, string>;
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
      const template = mergedMessages[key] ?? key;
      return interpolate(template, vars);
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
