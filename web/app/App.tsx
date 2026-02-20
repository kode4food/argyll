import React from "react";
import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import { Toaster } from "react-hot-toast";
import WebSocketProvider from "./contexts/WebSocketProvider";
import ConnectionStatusWrapper from "./components/atoms/ConnectionStatusWrapper";
import OverviewPage from "./components/templates/OverviewPage";
import LivePage from "./components/templates/LivePage";
import NotFoundPage from "./components/organisms/NotFoundPage";
import { I18nProvider } from "./i18n";
import enUS from "./i18n/en-US.json";
import deCH from "./i18n/de-CH.json";
import frCH from "./i18n/fr-CH.json";
import itCH from "./i18n/it-CH.json";
import { useLocale } from "./store/i18nStore";

const AUTOFILL_TARGET_SELECTOR =
  "input:not([type='password']), textarea, select";
const AUTOFILL_GUARDED_ATTR = "data-autofill-guarded";

const applyAutofillIgnore = (el: HTMLElement): void => {
  if (el.getAttribute(AUTOFILL_GUARDED_ATTR) === "true") {
    return;
  }
  el.setAttribute("autocomplete", "off");
  el.setAttribute("data-1p-ignore", "true");
  el.setAttribute("data-lpignore", "true");
  el.setAttribute("data-bwignore", "true");
  el.setAttribute(AUTOFILL_GUARDED_ATTR, "true");
};

type PluralForms = {
  zero?: string;
  one?: string;
  other: string;
};

type MessageValue = string | PluralForms;
type Messages = Record<string, MessageValue>;

const messagesByLocale: Record<string, Messages> = {
  "en-US": enUS as Messages,
  "de-CH": deCH as Messages,
  "fr-CH": frCH as Messages,
  "it-CH": itCH as Messages,
};

const LocaleProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const locale = useLocale();
  const messages = messagesByLocale[locale] ?? enUS;

  return (
    <I18nProvider locale={locale} messages={messages}>
      {children}
    </I18nProvider>
  );
};

const App: React.FC = () => {
  React.useEffect(() => {
    const handleInputInteraction = (event: Event): void => {
      if (!(event.target instanceof Element)) {
        return;
      }
      const target = event.target.closest<HTMLElement>(
        AUTOFILL_TARGET_SELECTOR
      );
      if (!target) {
        return;
      }
      applyAutofillIgnore(target);
    };

    document.addEventListener("mousedown", handleInputInteraction, true);
    document.addEventListener("focusin", handleInputInteraction, true);
    return () => {
      document.removeEventListener("mousedown", handleInputInteraction, true);
      document.removeEventListener("focusin", handleInputInteraction, true);
    };
  }, []);

  return (
    <Router>
      <LocaleProvider>
        <WebSocketProvider>
          <Routes>
            <Route path="/" element={<OverviewPage />} />
            <Route path="/flow/:flowId" element={<LivePage />} />
            <Route path="*" element={<NotFoundPage />} />
          </Routes>
          <ConnectionStatusWrapper />
        </WebSocketProvider>
      </LocaleProvider>
      <Toaster position="top-right" />
    </Router>
  );
};

export default App;
