import { create } from "zustand";
import { type User, UserManager, WebStorageStateStore } from "oidc-client-ts";
import { getEnv } from "@/utils/env";

interface AuthState {
  isEnabled: boolean;
  isReady: boolean;
  token: string | null;
  userName: string | null;
  start: () => Promise<void>;
  login: () => Promise<void>;
  logout: () => Promise<void>;
}

type SetAuth = (state: Partial<AuthState>) => void;

const OIDC_PROVIDER = "oidc";

const isEnabled = getEnv("AUTH_PROVIDER").trim() === OIDC_PROVIDER;

// Created only when login is enabled, so tests and open deployments never touch it
let manager: UserManager | null = null;

// A login callback can be completed once, even when React runs effects twice
let starting: Promise<void> | null = null;
let redirecting: Promise<void> | null = null;

const userManager = (): UserManager => {
  if (!manager) {
    const origin = window.location.origin;
    manager = new UserManager({
      authority: getEnv("AUTH_AUTHORITY"),
      client_id: getEnv("AUTH_CLIENT_ID"),
      redirect_uri: `${origin}/`,
      post_logout_redirect_uri: `${origin}/`,
      response_type: "code",
      scope: "openid",
      automaticSilentRenew: true,
      userStore: new WebStorageStateStore({ store: window.sessionStorage }),
    });
  }
  return manager;
};

const login = () => {
  const { pathname, search } = window.location;
  redirecting ??= userManager()
    .signinRedirect({ state: pathname + search })
    .catch((error: unknown) => {
      redirecting = null;
      throw error;
    });
  return redirecting;
};

const sessionOf = (user: User) => ({
  token: user.access_token,
  userName: String(user.profile.preferred_username ?? user.profile.sub),
});

// Finishes a login redirect or starts one, returning to the page that needed it
const startSession = async (set: SetAuth) => {
  const um = userManager();
  const params = new URLSearchParams(window.location.search);
  const isCallback = params.has("code") && params.has("state");
  const user = isCallback
    ? await um.signinRedirectCallback()
    : await um.getUser();
  if (isCallback) {
    const path = typeof user?.state === "string" ? user.state : "/";
    window.history.replaceState(null, "", path);
  }
  if (!user || user.expired) {
    await login();
    return;
  }
  um.events.addUserLoaded((next) => set(sessionOf(next)));
  um.events.addSilentRenewError(() => {
    void login().catch((error: unknown) =>
      console.error("Sign-in failed", error)
    );
  });
  set({ ...sessionOf(user), isReady: true });
};

const useAuthStore = create<AuthState>()((set) => ({
  isEnabled,
  isReady: !isEnabled,
  token: null,
  userName: null,
  start: () => {
    if (!isEnabled) return Promise.resolve();
    starting ??= startSession(set);
    return starting;
  },
  login,
  logout: async () => {
    set({ token: null, userName: null });
    await userManager().signoutRedirect();
  },
}));

export { useAuthStore };
