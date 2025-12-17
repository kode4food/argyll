export const getEnv = (key: string, defaultValue: string = ""): string => {
  if (typeof process !== "undefined" && process.env) {
    return process.env[`VITE_${key}`] || defaultValue;
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const viteEnv: any = (globalThis as any).__VITE_ENV__ || {};
  return viteEnv[key] || defaultValue;
};
