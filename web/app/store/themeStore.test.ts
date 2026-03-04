import { useThemeStore } from "./themeStore";

describe("themeStore", () => {
  beforeEach(() => {
    localStorage.clear();
    useThemeStore.setState({
      theme: "light",
    });
  });

  test("starts in light theme", () => {
    expect(useThemeStore.getState().theme).toBe("light");
  });

  test("setTheme updates theme directly", () => {
    useThemeStore.getState().setTheme("dark");
    expect(useThemeStore.getState().theme).toBe("dark");
  });

  test("toggleTheme switches between light and dark", () => {
    useThemeStore.getState().toggleTheme();
    expect(useThemeStore.getState().theme).toBe("dark");

    useThemeStore.getState().toggleTheme();
    expect(useThemeStore.getState().theme).toBe("light");
  });
});
