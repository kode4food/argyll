import { create } from "zustand";
import { devtools, persist, createJSONStorage } from "zustand/middleware";

export type Theme = "light" | "dark";

interface ThemeState {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
}

const defaultTheme: Theme = "light";

const useThemeStore = create<ThemeState>()(
  devtools(
    persist(
      (set) => ({
        theme: defaultTheme,
        setTheme: (theme) => set({ theme }, false, "theme/setTheme"),
        toggleTheme: () =>
          set(
            (state) => ({
              theme: state.theme === "dark" ? "light" : "dark",
            }),
            false,
            "theme/toggleTheme"
          ),
      }),
      {
        name: "themeStore",
        storage: createJSONStorage(() => localStorage),
      }
    ),
    { name: "themeStore" }
  )
);

const useTheme = () => useThemeStore((state) => state.theme);
const useToggleTheme = () => useThemeStore((state) => state.toggleTheme);

export { useThemeStore, useTheme, useToggleTheme };
