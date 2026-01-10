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

type Messages = Record<string, string>;

const messagesByLocale: Record<string, Messages> = {
  "en-US": enUS,
  "de-CH": deCH,
  "fr-CH": frCH,
  "it-CH": itCH,
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
