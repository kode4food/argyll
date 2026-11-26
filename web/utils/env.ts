export const getEnv = (key: string, defaultValue: string = ""): string => {
  if (typeof process !== "undefined" && process.env) {
    return process.env[`VITE_${key}`] || defaultValue;
  }

  // A deployed image sets its settings in env.js when it starts, ahead of the build's
  const runEnv: any = (globalThis as any).__ARGYLL_ENV__ || {};
  const viteEnv: any = (globalThis as any).__VITE_ENV__ || {};
  return runEnv[key] || viteEnv[key] || defaultValue;
};

export const isDevServer = (globalThis as any).__DEV_SERVER__ === true;
