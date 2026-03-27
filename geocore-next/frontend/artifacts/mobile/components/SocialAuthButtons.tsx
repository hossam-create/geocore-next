import * as AppleAuthentication from "expo-apple-authentication";
import * as Google from "expo-auth-session/providers/google";
import * as WebBrowser from "expo-web-browser";
import * as Haptics from "expo-haptics";
import { Platform, Pressable, StyleSheet, Text, View, ActivityIndicator, Alert } from "react-native";
import { useEffect, useState } from "react";
import { useColorScheme } from "react-native";
import Colors from "@/constants/colors";
import { useAuthStore } from "@/store/authStore";
import { router } from "expo-router";

WebBrowser.maybeCompleteAuthSession();

const GOOGLE_CLIENT_ID = process.env.EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID ?? "";
const GOOGLE_IOS_CLIENT_ID = process.env.EXPO_PUBLIC_GOOGLE_IOS_CLIENT_ID ?? "";
const GOOGLE_ANDROID_CLIENT_ID = process.env.EXPO_PUBLIC_GOOGLE_ANDROID_CLIENT_ID ?? "";

interface Props {
  mode: "login" | "register";
}

export default function SocialAuthButtons({ mode }: Props) {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const { socialLogin, isLoading } = useAuthStore();
  const [busyProvider, setBusyProvider] = useState<string | null>(null);

  const [googleRequest, googleResponse, googlePrompt] = Google.useAuthRequest({
    webClientId: GOOGLE_CLIENT_ID,
    iosClientId: GOOGLE_IOS_CLIENT_ID,
    androidClientId: GOOGLE_ANDROID_CLIENT_ID,
    scopes: ["profile", "email"],
  });

  useEffect(() => {
    if (googleResponse?.type === "success") {
      const token = googleResponse.authentication?.accessToken;
      if (token) {
        handleSocialToken("google", token);
      }
    } else if (googleResponse?.type === "error") {
      setBusyProvider(null);
      Alert.alert("Google Sign-In Failed", "Could not authenticate with Google. Please try again.");
    } else if (googleResponse?.type === "cancel" || googleResponse?.type === "dismiss") {
      setBusyProvider(null);
    }
  }, [googleResponse]);

  const handleSocialToken = async (
    provider: "google" | "apple" | "facebook",
    token: string,
    name?: string,
    email?: string
  ) => {
    setBusyProvider(provider);
    try {
      await socialLogin(provider, token, name, email);
      router.replace("/(tabs)");
    } catch (err: any) {
      const msg = err?.response?.data?.message ?? err?.message ?? `${provider} sign-in failed.`;
      Alert.alert("Authentication Failed", msg);
    } finally {
      setBusyProvider(null);
    }
  };

  const handleGoogle = async () => {
    if (!GOOGLE_CLIENT_ID && !GOOGLE_IOS_CLIENT_ID && !GOOGLE_ANDROID_CLIENT_ID) {
      Alert.alert(
        "Google Sign-In",
        "Google OAuth credentials are not configured yet. Please set EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID in your environment."
      );
      return;
    }
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    setBusyProvider("google");
    await googlePrompt();
  };

  const handleApple = async () => {
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    setBusyProvider("apple");
    try {
      const credential = await AppleAuthentication.signInAsync({
        requestedScopes: [
          AppleAuthentication.AppleAuthenticationScope.FULL_NAME,
          AppleAuthentication.AppleAuthenticationScope.EMAIL,
        ],
      });
      const fullName = credential.fullName
        ? [credential.fullName.givenName, credential.fullName.familyName]
            .filter(Boolean)
            .join(" ")
        : undefined;
      await handleSocialToken(
        "apple",
        credential.identityToken!,
        fullName || undefined,
        credential.email || undefined
      );
    } catch (err: any) {
      if (err?.code !== "ERR_REQUEST_CANCELED" && err?.code !== "1001") {
        Alert.alert("Apple Sign-In Failed", err?.message ?? "Could not authenticate with Apple.");
      }
      setBusyProvider(null);
    }
  };

  const handleFacebook = async () => {
    Alert.alert(
      "Facebook Sign-In",
      "Facebook login coming soon. Please use Google or Apple for now."
    );
  };

  const isAppleAvailable = Platform.OS === "ios";

  return (
    <View style={styles.container}>
      <View style={styles.divider}>
        <View style={[styles.dividerLine, { backgroundColor: colors.border }]} />
        <Text style={[styles.dividerText, { color: colors.textTertiary }]}>or continue with</Text>
        <View style={[styles.dividerLine, { backgroundColor: colors.border }]} />
      </View>

      <View style={styles.row}>
        <SocialButton
          label="Google"
          icon={<GoogleIcon />}
          onPress={handleGoogle}
          busy={busyProvider === "google"}
          disabled={!!busyProvider && busyProvider !== "google"}
          bg={colorScheme === "dark" ? "#1E1E1E" : "#fff"}
          borderColor={colors.border}
          textColor={colors.text}
        />
        {isAppleAvailable && (
          <SocialButton
            label="Apple"
            icon={<AppleIcon dark={colorScheme === "dark"} />}
            onPress={handleApple}
            busy={busyProvider === "apple"}
            disabled={!!busyProvider && busyProvider !== "apple"}
            bg={colorScheme === "dark" ? "#fff" : "#000"}
            borderColor={colorScheme === "dark" ? "#fff" : "#000"}
            textColor={colorScheme === "dark" ? "#000" : "#fff"}
          />
        )}
        <SocialButton
          label="Facebook"
          icon={<FacebookIcon />}
          onPress={handleFacebook}
          busy={busyProvider === "facebook"}
          disabled={!!busyProvider && busyProvider !== "facebook"}
          bg="#1877F2"
          borderColor="#1877F2"
          textColor="#fff"
        />
      </View>
    </View>
  );
}

function SocialButton({
  label,
  icon,
  onPress,
  busy,
  disabled,
  bg,
  borderColor,
  textColor,
}: {
  label: string;
  icon: React.ReactNode;
  onPress: () => void;
  busy: boolean;
  disabled: boolean;
  bg: string;
  borderColor: string;
  textColor: string;
}) {
  return (
    <Pressable
      onPress={onPress}
      disabled={disabled || busy}
      style={({ pressed }) => [
        styles.socialBtn,
        { backgroundColor: bg, borderColor, opacity: pressed || disabled ? 0.7 : 1 },
      ]}
    >
      {busy ? (
        <ActivityIndicator size="small" color={textColor} />
      ) : (
        <>
          {icon}
          <Text style={[styles.socialBtnText, { color: textColor }]}>{label}</Text>
        </>
      )}
    </Pressable>
  );
}

function GoogleIcon() {
  return (
    <Text style={styles.iconText}>G</Text>
  );
}

function AppleIcon({ dark }: { dark: boolean }) {
  return (
    <Text style={[styles.iconText, { color: dark ? "#000" : "#fff", fontSize: 18 }]}>🍎</Text>
  );
}

function FacebookIcon() {
  return (
    <Text style={[styles.iconText, { color: "#fff", fontFamily: "Inter_700Bold" }]}>f</Text>
  );
}

const styles = StyleSheet.create({
  container: {
    gap: 14,
  },
  divider: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
  },
  dividerLine: {
    flex: 1,
    height: 1,
  },
  dividerText: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
    textAlign: "center",
  },
  row: {
    flexDirection: "row",
    gap: 10,
  },
  socialBtn: {
    flex: 1,
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: 7,
    borderRadius: 10,
    borderWidth: 1.5,
    height: 48,
    paddingHorizontal: 8,
  },
  socialBtnText: {
    fontSize: 13,
    fontFamily: "Inter_600SemiBold",
  },
  iconText: {
    fontSize: 16,
    fontFamily: "Inter_700Bold",
    color: "#EA4335",
  },
});
