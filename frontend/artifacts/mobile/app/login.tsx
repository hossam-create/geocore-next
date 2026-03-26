import { Feather } from "@expo/vector-icons";
import * as Haptics from "expo-haptics";
import { router } from "expo-router";
import React, { useState } from "react";
import {
  Alert,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
  useColorScheme,
} from "react-native";

import Colors from "@/constants/colors";
import { useAuthStore } from "@/store/authStore";
import SocialAuthButtons from "@/components/SocialAuthButtons";

export default function LoginScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const { login, isLoading } = useAuthStore();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);

  const handleLogin = async () => {
    if (!email.trim() || !password.trim()) {
      Alert.alert("Missing fields", "Please enter your email and password.");
      return;
    }
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
    try {
      await login(email.trim(), password);
      router.replace("/(tabs)");
    } catch (err: any) {
      Alert.alert(
        "Login Failed",
        err?.response?.data?.message ?? "Invalid email or password."
      );
    }
  };

  return (
    <KeyboardAvoidingView
      style={[styles.root, { backgroundColor: colors.background }]}
      behavior={Platform.OS === "ios" ? "padding" : "height"}
    >
      <ScrollView
        contentContainerStyle={styles.scroll}
        keyboardShouldPersistTaps="handled"
        showsVerticalScrollIndicator={false}
      >
        <View style={styles.headerBlock}>
          <Text style={styles.logo}>GeoCore</Text>
          <Text style={[styles.tagline, { color: colors.textSecondary }]}>
            The GCC Marketplace
          </Text>
        </View>

        <View
          style={[
            styles.card,
            {
              backgroundColor: colors.backgroundSecondary,
              borderColor: colors.border,
            },
          ]}
        >
          <Text style={[styles.cardTitle, { color: colors.text }]}>
            Sign in
          </Text>

          <SocialAuthButtons mode="login" />

          <View style={styles.divider}>
            <View style={[styles.dividerLine, { backgroundColor: colors.border }]} />
            <Text style={[styles.dividerText, { color: colors.textTertiary }]}>
              or use email
            </Text>
            <View style={[styles.dividerLine, { backgroundColor: colors.border }]} />
          </View>

          <View style={styles.field}>
            <Text style={[styles.label, { color: colors.textSecondary }]}>
              Email
            </Text>
            <View
              style={[
                styles.inputWrap,
                {
                  backgroundColor: colors.backgroundTertiary,
                  borderColor: colors.border,
                },
              ]}
            >
              <Feather name="mail" size={16} color={colors.textTertiary} />
              <TextInput
                style={[styles.input, { color: colors.text }]}
                placeholder="you@example.com"
                placeholderTextColor={colors.textTertiary}
                value={email}
                onChangeText={setEmail}
                keyboardType="email-address"
                autoCapitalize="none"
                autoCorrect={false}
                returnKeyType="next"
              />
            </View>
          </View>

          <View style={styles.field}>
            <Text style={[styles.label, { color: colors.textSecondary }]}>
              Password
            </Text>
            <View
              style={[
                styles.inputWrap,
                {
                  backgroundColor: colors.backgroundTertiary,
                  borderColor: colors.border,
                },
              ]}
            >
              <Feather name="lock" size={16} color={colors.textTertiary} />
              <TextInput
                style={[styles.input, { color: colors.text }]}
                placeholder="••••••••"
                placeholderTextColor={colors.textTertiary}
                value={password}
                onChangeText={setPassword}
                secureTextEntry={!showPassword}
                returnKeyType="done"
                onSubmitEditing={handleLogin}
              />
              <Pressable onPress={() => setShowPassword((v) => !v)}>
                <Feather
                  name={showPassword ? "eye-off" : "eye"}
                  size={16}
                  color={colors.textTertiary}
                />
              </Pressable>
            </View>
          </View>

          <Pressable
            onPress={handleLogin}
            disabled={isLoading}
            style={({ pressed }) => [
              styles.primaryBtn,
              { opacity: pressed || isLoading ? 0.8 : 1 },
            ]}
          >
            <Text style={styles.primaryBtnText}>
              {isLoading ? "Signing in…" : "Sign In"}
            </Text>
          </Pressable>

          <Pressable
            onPress={() => router.push("/register")}
            style={({ pressed }) => [
              styles.secondaryBtn,
              {
                borderColor: "#0071CE",
                opacity: pressed ? 0.8 : 1,
              },
            ]}
          >
            <Text style={[styles.secondaryBtnText, { color: "#0071CE" }]}>
              Create Account
            </Text>
          </Pressable>
        </View>

        <Pressable
          onPress={() => router.replace("/(tabs)")}
          style={({ pressed }) => [styles.guestBtn, pressed && { opacity: 0.7 }]}
        >
          <Text style={[styles.guestText, { color: colors.textTertiary }]}>
            Browse as guest →
          </Text>
        </Pressable>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1 },
  scroll: {
    flexGrow: 1,
    paddingHorizontal: 24,
    paddingTop: 80,
    paddingBottom: 40,
  },
  headerBlock: {
    alignItems: "center",
    marginBottom: 36,
  },
  logo: {
    fontSize: 36,
    fontFamily: "Inter_700Bold",
    color: "#0071CE",
  },
  tagline: {
    fontSize: 14,
    fontFamily: "Inter_400Regular",
    marginTop: 4,
  },
  card: {
    borderRadius: 16,
    padding: 24,
    borderWidth: 1,
    gap: 16,
  },
  cardTitle: {
    fontSize: 22,
    fontFamily: "Inter_700Bold",
    marginBottom: 4,
  },
  field: {
    gap: 6,
  },
  label: {
    fontSize: 13,
    fontFamily: "Inter_500Medium",
  },
  inputWrap: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    borderRadius: 10,
    borderWidth: 1,
    paddingHorizontal: 14,
    height: 50,
  },
  input: {
    flex: 1,
    fontSize: 15,
    fontFamily: "Inter_400Regular",
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
  },
  primaryBtn: {
    backgroundColor: "#0071CE",
    borderRadius: 10,
    height: 52,
    alignItems: "center",
    justifyContent: "center",
    marginTop: 4,
  },
  primaryBtnText: {
    color: "#fff",
    fontSize: 16,
    fontFamily: "Inter_600SemiBold",
  },
  secondaryBtn: {
    borderWidth: 1.5,
    borderRadius: 10,
    height: 52,
    alignItems: "center",
    justifyContent: "center",
  },
  secondaryBtnText: {
    fontSize: 16,
    fontFamily: "Inter_600SemiBold",
  },
  guestBtn: {
    alignItems: "center",
    paddingVertical: 20,
  },
  guestText: {
    fontSize: 14,
    fontFamily: "Inter_400Regular",
  },
});
