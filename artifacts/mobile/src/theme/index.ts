import { darkColors, lightColors, type ColorPalette } from "./colors";
import { shadows } from "./shadows";
import { radius, spacing } from "./spacing";
import { typography, fontFamily, fontSize } from "./typography";

export type ThemeMode = "light" | "dark";

export interface Theme {
  mode: ThemeMode;
  colors: ColorPalette;
  spacing: typeof spacing;
  radius: typeof radius;
  shadows: typeof shadows;
  typography: typeof typography;
  fontFamily: typeof fontFamily;
  fontSize: typeof fontSize;
}

export const lightTheme: Theme = {
  mode: "light",
  colors: lightColors,
  spacing,
  radius,
  shadows,
  typography,
  fontFamily,
  fontSize,
};

export const darkTheme: Theme = {
  mode: "dark",
  colors: darkColors,
  spacing,
  radius,
  shadows,
  typography,
  fontFamily,
  fontSize,
};

export function getTheme(mode: ThemeMode): Theme {
  return mode === "dark" ? darkTheme : lightTheme;
}

export { darkColors, lightColors } from "./colors";
export { spacing, radius } from "./spacing";
export { shadows } from "./shadows";
export { typography, fontFamily, fontSize } from "./typography";
export type { ColorPalette } from "./colors";
export type { SpacingKey, RadiusKey } from "./spacing";
export type { ShadowKey } from "./shadows";
export type { TypographyKey } from "./typography";
