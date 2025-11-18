import type { Metadata } from "next";
import "./index.css";
import { WebSocketProvider } from "./hooks/useWebSocketContext";
import React from "react";
import { Toaster } from "react-hot-toast";
import ConnectionStatusWrapper from "./components/atoms/ConnectionStatusWrapper";

export const metadata: Metadata = {
  title: "Spuds",
  description: "Spuds Flow Processing System",
  icons: {
    icon: "/favicon.ico",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>
        <noscript>You need to enable JavaScript to run this app.</noscript>
        <WebSocketProvider>
          {children}
          <ConnectionStatusWrapper />
        </WebSocketProvider>
        <Toaster position="top-right" />
      </body>
    </html>
  );
}
