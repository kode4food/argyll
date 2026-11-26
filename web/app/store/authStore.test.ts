type AuthStoreModule = typeof import("./authStore");

const mockManager = {
  getUser: jest.fn(),
  signinRedirect: jest.fn(),
  signinRedirectCallback: jest.fn(),
  signoutRedirect: jest.fn(),
  signinSilent: jest.fn(),
  events: {
    addUserLoaded: jest.fn(),
    addSilentRenewError: jest.fn(),
    addAccessTokenExpired: jest.fn(),
  },
};

jest.mock("oidc-client-ts", () => ({
  UserManager: jest.fn(() => mockManager),
  WebStorageStateStore: jest.fn(),
}));

const user = (expired = false) => ({
  access_token: "token-1",
  expired,
  state: undefined as unknown,
  profile: { sub: "sub-1", preferred_username: "operator" },
});

// The store decides at load time whether login is on, so each test loads it fresh
const loadStore = (provider?: string): AuthStoreModule => {
  if (provider) {
    process.env.VITE_AUTH_PROVIDER = provider;
  } else {
    delete process.env.VITE_AUTH_PROVIDER;
  }
  let mod: AuthStoreModule | undefined;
  jest.isolateModules(() => {
    mod = require("./authStore");
  });
  return mod!;
};

describe("authStore", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockManager.signinRedirect.mockResolvedValue(undefined);
    window.history.replaceState(null, "", "/flow/f1");
  });

  afterAll(() => {
    delete process.env.VITE_AUTH_PROVIDER;
  });

  test("stays ready without login", async () => {
    const { useAuthStore } = loadStore();
    await useAuthStore.getState().start();

    expect(useAuthStore.getState()).toMatchObject({
      isEnabled: false,
      isReady: true,
      token: null,
    });
    expect(mockManager.getUser).not.toHaveBeenCalled();
  });

  test("uses a signed-in user", async () => {
    mockManager.getUser.mockResolvedValue(user());
    const { useAuthStore } = loadStore("oidc");
    expect(useAuthStore.getState().isReady).toBe(false);

    await useAuthStore.getState().start();
    await useAuthStore.getState().start();

    expect(mockManager.getUser).toHaveBeenCalledTimes(1);
    expect(useAuthStore.getState()).toMatchObject({
      isReady: true,
      token: "token-1",
      userName: "operator",
    });

    const loaded = mockManager.events.addUserLoaded.mock.calls[0][0];
    loaded({ ...user(), access_token: "token-2" });
    expect(useAuthStore.getState().token).toBe("token-2");
  });

  test("renews an expired token, or signs in when renewal fails", async () => {
    mockManager.getUser.mockResolvedValue({ ...user(), expires_at: 100 });
    const { useAuthStore } = loadStore("oidc");
    await useAuthStore.getState().start();
    expect(useAuthStore.getState().expiresAt).toBe(100000);

    const expired = mockManager.events.addAccessTokenExpired.mock.calls[0][0];
    mockManager.signinSilent.mockResolvedValueOnce(undefined);
    await expired();
    expect(mockManager.signinRedirect).not.toHaveBeenCalled();

    mockManager.signinSilent.mockRejectedValueOnce(new Error("no session"));
    expired();
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(mockManager.signinRedirect).toHaveBeenCalled();
  });

  test("redirects to sign in, returning to the current page", async () => {
    mockManager.getUser.mockResolvedValue(user(true));
    const { useAuthStore } = loadStore("oidc");

    await useAuthStore.getState().start();

    expect(mockManager.signinRedirect).toHaveBeenCalledWith({
      state: "/flow/f1",
    });
    expect(useAuthStore.getState().isReady).toBe(false);
  });

  test("shares a pending sign-in redirect", async () => {
    const { useAuthStore } = loadStore("oidc");
    const { login } = useAuthStore.getState();

    await Promise.all([login(), login()]);

    expect(mockManager.signinRedirect).toHaveBeenCalledTimes(1);
  });

  test("completes a sign-in callback", async () => {
    window.history.replaceState(null, "", "/?code=c&state=s");
    mockManager.signinRedirectCallback.mockResolvedValue({
      ...user(),
      state: "/flow/f2",
    });
    mockManager.getUser.mockResolvedValue(user());
    const { useAuthStore } = loadStore("oidc");

    await useAuthStore.getState().start();

    expect(mockManager.signinRedirectCallback).toHaveBeenCalled();
    expect(window.location.pathname).toBe("/flow/f2");
    expect(useAuthStore.getState().isReady).toBe(true);
  });

  test("signs out", async () => {
    mockManager.getUser.mockResolvedValue(user());
    const { useAuthStore } = loadStore("oidc");
    await useAuthStore.getState().start();

    await useAuthStore.getState().logout();

    expect(useAuthStore.getState().token).toBeNull();
    expect(mockManager.signoutRedirect).toHaveBeenCalled();
  });
});
