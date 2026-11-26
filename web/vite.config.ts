import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

const ENV_PREFIX = "VITE_";

// getEnv reads VITE_ settings from globalThis.__VITE_ENV__, keyed without the prefix
const browserEnv = (mode: string) =>
  Object.fromEntries(
    Object.entries(loadEnv(mode, process.cwd(), ENV_PREFIX)).map(([k, v]) => [
      k.slice(ENV_PREFIX.length),
      v,
    ])
  );

export default defineConfig(({ command, mode }) => ({
  define: {
    "globalThis.__VITE_ENV__": JSON.stringify(browserEnv(mode)),
    "globalThis.__DEV_SERVER__": JSON.stringify(command === "serve"),
  },
  plugins: [react()],
  server: {
    port: 3001,
    host: true,
    // Outside Compose, the dev server forwards the backend paths to the local stack
    proxy: Object.fromEntries(
      ["/engine", "/chat", "/auth", "/health"].map((path) => [
        path,
        { target: "http://localhost:8080", ws: true },
      ])
    ),
  },
  build: {
    outDir: "dist",
    sourcemap: false,
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./"),
    },
  },
}));
