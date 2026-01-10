import { create } from "zustand";
import { devtools } from "zustand/middleware";

const supportedLocales = ["en-US", "de-CH", "fr-CH", "it-CH"] as const;
type Locale = (typeof supportedLocales)[number];

interface I18nState {
  locale: Locale;
  setLocale: (locale: Locale) => void;
}

declare global {
  interface Window {
    i18nStore?: typeof useI18nStore;
  }
}

const useI18nStore = create<I18nState>()(
  devtools(
    (set) => ({
      locale: "en-US",
      setLocale: (locale) => set({ locale }, false, "i18n/setLocale"),
    }),
    { name: "i18nStore" }
  )
);

const isDevHost =
  typeof window !== "undefined" &&
  (window.location.hostname === "localhost" ||
    window.location.hostname === "127.0.0.1");

if (isDevHost) {
  window.i18nStore = useI18nStore;
}

const useLocale = () => useI18nStore((state) => state.locale);
const useSetLocale = () => useI18nStore((state) => state.setLocale);

export { supportedLocales, useI18nStore, useLocale, useSetLocale };
