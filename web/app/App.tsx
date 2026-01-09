import React from "react";
import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import { Toaster } from "react-hot-toast";
import WebSocketProvider from "./contexts/WebSocketProvider";
import ConnectionStatusWrapper from "./components/atoms/ConnectionStatusWrapper";
import OverviewPage from "./components/templates/OverviewPage";
import LivePage from "./components/templates/LivePage";
import NotFoundPage from "./components/organisms/NotFoundPage";

const App: React.FC = () => {
  return (
    <Router>
      <WebSocketProvider>
        <Routes>
          <Route path="/" element={<OverviewPage />} />
          <Route path="/flow/:flowId" element={<LivePage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Routes>
        <ConnectionStatusWrapper />
      </WebSocketProvider>
      <Toaster position="top-right" />
    </Router>
  );
};

export default App;
