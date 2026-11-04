export const getEnv = (key: string, defaultValue: string = ""): string => {
  if (typeof process !== "undefined" && process.env) {
    return process.env[`VITE_${key}`] || defaultValue;
  }

  const viteEnv: any = (globalThis as any).__VITE_ENV__ || {};
  return viteEnv[key] || defaultValue;
};

export const isDevHost =
  typeof window !== "undefined" &&
  (window.location.hostname === "localhost" ||
    window.location.hostname === "127.0.0.1");
