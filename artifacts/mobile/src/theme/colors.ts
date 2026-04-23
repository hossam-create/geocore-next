/**
 * Theme colors — split into light/dark palettes. Keep semantic tokens
 * stable across modes so components don't branch on mode.
 */
interface BrandColors {
  readonly blue: string;
  readonly blueDark: string;
  readonly yellow: string;
  readonly yellowDark: string;
  readonly red: string;
  readonly green: string;
}

const brand: BrandColors = {
  blue: "#0071CE",
  blueDark: "#005BA1",
  yellow: "#FFC220",
  yellowDark: "#E5AC00",
  red: "#E53935",
  green: "#4CAF50",
};

export interface ColorPalette {
  readonly brand: BrandColors;
  readonly text: string;
  readonly textSecondary: string;
  readonly textTertiary: string;
  readonly textInverse: string;
  readonly background: string;
  readonly surface: string;
  readonly surfaceElevated: string;
  readonly surfaceMuted: string;
  readonly border: string;
  readonly borderLight: string;
  readonly primary: string;
  readonly primaryDark: string;
  readonly secondary: string;
  readonly secondaryDark: string;
  readonly success: string;
  readonly warning: string;
  readonly error: string;
  readonly info: string;
  readonly auctionBadge: string;
  readonly buyNowBadge: string;
  readonly featuredBadge: string;
  readonly overlay: string;
}

export const lightColors: ColorPalette = {
  brand,
  text: "#1A1A1A",
  textSecondary: "#444444",
  textTertiary: "#888888",
  textInverse: "#FFFFFF",
  background: "#F5F5F5",
  surface: "#FFFFFF",
  surfaceElevated: "#FFFFFF",
  surfaceMuted: "#E8E8E8",
  border: "#E0E0E0",
  borderLight: "#EFEFEF",
  primary: brand.blue,
  primaryDark: brand.blueDark,
  secondary: brand.yellow,
  secondaryDark: brand.yellowDark,
  success: brand.green,
  warning: brand.yellow,
  error: brand.red,
  info: brand.blue,
  auctionBadge: brand.red,
  buyNowBadge: brand.blue,
  featuredBadge: brand.yellow,
  overlay: "rgba(0, 0, 0, 0.5)",
};

export const darkColors: ColorPalette = {
  brand,
  text: "#F5F5F5",
  textSecondary: "#BBBBBB",
  textTertiary: "#888888",
  textInverse: "#1A1A1A",
  background: "#121212",
  surface: "#1E1E1E",
  surfaceElevated: "#242424",
  surfaceMuted: "#2A2A2A",
  border: "#333333",
  borderLight: "#2A2A2A",
  primary: brand.blue,
  primaryDark: brand.blueDark,
  secondary: brand.yellow,
  secondaryDark: brand.yellowDark,
  success: brand.green,
  warning: brand.yellow,
  error: brand.red,
  info: brand.blue,
  auctionBadge: brand.red,
  buyNowBadge: brand.blue,
  featuredBadge: brand.yellow,
  overlay: "rgba(0, 0, 0, 0.7)",
};
