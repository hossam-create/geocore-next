import {
  Inter_400Regular,
  Inter_500Medium,
  Inter_600SemiBold,
  Inter_700Bold,
  useFonts,
} from "@expo-google-fonts/inter";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Stack } from "expo-router";
import * as SplashScreen from "expo-splash-screen";
import React, { useEffect } from "react";
import { GestureHandlerRootView } from "react-native-gesture-handler";
import { KeyboardProvider } from "react-native-keyboard-controller";
import { SafeAreaProvider } from "react-native-safe-area-context";

import { ErrorBoundary } from "@/components/ErrorBoundary";
import { AppProvider } from "@/context/AppContext";
import { useAuthStore } from "@/store/authStore";

SplashScreen.preventAutoHideAsync();

const queryClient = new QueryClient();

function RootLayoutNav() {
  return (
    <Stack screenOptions={{ headerShown: false }}>
      <Stack.Screen name="(tabs)" />
      <Stack.Screen
        name="login"
        options={{ presentation: "modal", animation: "slide_from_bottom" }}
      />
      <Stack.Screen
        name="register"
        options={{ presentation: "card", animation: "slide_from_right" }}
      />
      <Stack.Screen
        name="listing/[id]"
        options={{ presentation: "card", animation: "slide_from_right" }}
      />
      <Stack.Screen
        name="conversation/[id]"
        options={{ presentation: "card", animation: "slide_from_right" }}
      />
      <Stack.Screen
        name="favorites"
        options={{ presentation: "card", animation: "slide_from_right" }}
      />
      <Stack.Screen
        name="my-listings"
        options={{ presentation: "card", animation: "slide_from_right" }}
      />
      <Stack.Screen
        name="my-bids"
        options={{ presentation: "card", animation: "slide_from_right" }}
      />
      <Stack.Screen
        name="wallet"
        options={{ presentation: "card", animation: "slide_from_right" }}
      />
      <Stack.Screen
        name="notifications"
        options={{ presentation: "card", animation: "slide_from_right" }}
      />
      <Stack.Screen
        name="settings"
        options={{ presentation: "card", animation: "slide_from_right" }}
      />
      <Stack.Screen
        name="help"
        options={{ presentation: "card", animation: "slide_from_right" }}
      />
      <Stack.Screen
        name="reviews/[userId]"
        options={{ presentation: "card", animation: "slide_from_right" }}
      />
    </Stack>
  );
}

function AppWithSession() {
  const restoreSession = useAuthStore((s) => s.restoreSession);

  useEffect(() => {
    restoreSession();
  }, []);

  return (
    <AppProvider>
      <GestureHandlerRootView>
        <KeyboardProvider>
          <RootLayoutNav />
        </KeyboardProvider>
      </GestureHandlerRootView>
    </AppProvider>
  );
}

export default function RootLayout() {
  const [fontsLoaded, fontError] = useFonts({
    Inter_400Regular,
    Inter_500Medium,
    Inter_600SemiBold,
    Inter_700Bold,
  });

  useEffect(() => {
    if (fontsLoaded || fontError) {
      SplashScreen.hideAsync();
    }
  }, [fontsLoaded, fontError]);

  if (!fontsLoaded && !fontError) return null;

  return (
    <SafeAreaProvider>
      <ErrorBoundary>
        <QueryClientProvider client={queryClient}>
          <AppWithSession />
        </QueryClientProvider>
      </ErrorBoundary>
    </SafeAreaProvider>
  );
}
