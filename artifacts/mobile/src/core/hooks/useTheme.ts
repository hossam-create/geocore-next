import { useColorScheme } from "react-native";

import { getTheme, type Theme, type ThemeMode } from "../../theme";

export function useTheme(): Theme {
  const scheme = useColorScheme();
  const mode: ThemeMode = scheme === "dark" ? "dark" : "light";
  return getTheme(mode);
}
