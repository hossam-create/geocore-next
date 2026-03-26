import { Platform } from "react-native";
import { notificationsAPI } from "./api";

/**
 * Registers an Expo push token with the GeoCore backend.
 * Should be called after the user logs in.
 *
 * This is a no-op on web since Expo push notifications only work on native.
 */
export async function registerPushTokenWithBackend(expoPushToken: string): Promise<void> {
  if (!expoPushToken) return;

  const platform = Platform.OS === "ios" ? "ios" : Platform.OS === "android" ? "android" : "web";

  try {
    await notificationsAPI.registerPushToken(expoPushToken, platform);
  } catch (err) {
    // Non-fatal — user will still receive in-app and WebSocket notifications
    console.warn("[push] Failed to register push token:", err);
  }
}

/**
 * Requests permission and retrieves the Expo push token.
 * Returns null if permission is denied or running on web.
 */
export async function getExpoPushToken(): Promise<string | null> {
  if (Platform.OS === "web") return null;

  try {
    // Dynamic import to avoid web bundle issues
    const Notifications = await import("expo-notifications");
    const { status: existingStatus } = await Notifications.getPermissionsAsync();
    let finalStatus = existingStatus;

    if (existingStatus !== "granted") {
      const { status } = await Notifications.requestPermissionsAsync();
      finalStatus = status;
    }

    if (finalStatus !== "granted") {
      console.log("[push] Push notification permission denied");
      return null;
    }

    const tokenData = await Notifications.getExpoPushTokenAsync();
    return tokenData.data;
  } catch (err) {
    console.warn("[push] Could not get Expo push token:", err);
    return null;
  }
}

/**
 * Full setup: request permission → get token → register with backend.
 * Call this after user successfully authenticates.
 */
export async function setupPushNotifications(): Promise<void> {
  if (Platform.OS === "web") return;

  const token = await getExpoPushToken();
  if (token) {
    await registerPushTokenWithBackend(token);
  }
}
