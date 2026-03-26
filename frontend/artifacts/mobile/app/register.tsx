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

export default function RegisterScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const { register: signUp, isLoading } = useAuthStore();

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [showPassword, setShowPassword] = useState(false);

  const handleRegister = async () => {
    if (!name.trim() || !email.trim() || !password) {
      Alert.alert("Missing fields", "Name, email and password are required.");
      return;
    }
    if (password !== confirm) {
      Alert.alert("Password mismatch", "Passwords do not match.");
      return;
    }
    if (password.length < 8) {
      Alert.alert("Weak password", "Password must be at least 8 characters.");
      return;
    }
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
    try {
      await signUp(name.trim(), email.trim(), password, phone.trim() || undefined);
      router.replace("/(tabs)");
    } catch (err: any) {
      Alert.alert(
        "Registration Failed",
        err?.response?.data?.message ?? "Could not create account. Try again."
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
            Create your account
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
            Join GeoCore
          </Text>

          <SocialAuthButtons mode="register" />

          <View style={styles.divider}>
            <View style={[styles.dividerLine, { backgroundColor: colors.border }]} />
            <Text style={[styles.dividerText, { color: colors.textTertiary }]}>
              or register with email
            </Text>
            <View style={[styles.dividerLine, { backgroundColor: colors.border }]} />
          </View>

          {[
            { label: "Full Name", icon: "user", value: name, setter: setName, placeholder: "Ahmed Al-Rashid", type: "default" },
            { label: "Email", icon: "mail", value: email, setter: setEmail, placeholder: "you@example.com", type: "email-address" },
            { label: "Phone (optional)", icon: "phone", value: phone, setter: setPhone, placeholder: "+971 50 000 0000", type: "phone-pad" },
          ].map((field) => (
            <View key={field.label} style={styles.field}>
              <Text style={[styles.label, { color: colors.textSecondary }]}>
                {field.label}
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
                <Feather name={field.icon as any} size={16} color={colors.textTertiary} />
                <TextInput
                  style={[styles.input, { color: colors.text }]}
                  placeholder={field.placeholder}
                  placeholderTextColor={colors.textTertiary}
                  value={field.value}
                  onChangeText={field.setter}
                  keyboardType={field.type as any}
                  autoCapitalize={field.type === "default" ? "words" : "none"}
                  autoCorrect={false}
                  returnKeyType="next"
                />
              </View>
            </View>
          ))}

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
                placeholder="Min. 8 characters"
                placeholderTextColor={colors.textTertiary}
                value={password}
                onChangeText={setPassword}
                secureTextEntry={!showPassword}
                returnKeyType="next"
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

          <View style={styles.field}>
            <Text style={[styles.label, { color: colors.textSecondary }]}>
              Confirm Password
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
                placeholder="Repeat password"
                placeholderTextColor={colors.textTertiary}
                value={confirm}
                onChangeText={setConfirm}
                secureTextEntry={!showPassword}
                returnKeyType="done"
                onSubmitEditing={handleRegister}
              />
            </View>
          </View>

          <Pressable
            onPress={handleRegister}
            disabled={isLoading}
            style={({ pressed }) => [
              styles.primaryBtn,
              { opacity: pressed || isLoading ? 0.8 : 1 },
            ]}
          >
            <Text style={styles.primaryBtnText}>
              {isLoading ? "Creating account…" : "Create Account"}
            </Text>
          </Pressable>

          <Pressable
            onPress={() => router.back()}
            style={({ pressed }) => [
              styles.secondaryBtn,
              { borderColor: "#0071CE", opacity: pressed ? 0.8 : 1 },
            ]}
          >
            <Text style={[styles.secondaryBtnText, { color: "#0071CE" }]}>
              Already have an account? Sign In
            </Text>
          </Pressable>
        </View>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1 },
  scroll: {
    flexGrow: 1,
    paddingHorizontal: 24,
    paddingTop: 70,
    paddingBottom: 40,
  },
  headerBlock: {
    alignItems: "center",
    marginBottom: 28,
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
    gap: 14,
  },
  cardTitle: {
    fontSize: 22,
    fontFamily: "Inter_700Bold",
    marginBottom: 4,
  },
  field: { gap: 6 },
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
    paddingHorizontal: 12,
  },
  secondaryBtnText: {
    fontSize: 14,
    fontFamily: "Inter_600SemiBold",
    textAlign: "center",
  },
});
