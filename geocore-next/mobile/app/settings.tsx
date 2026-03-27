import { Feather } from "@expo/vector-icons";
import { router } from "expo-router";
import React, { useState } from "react";
import {
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Switch,
  Text,
  View,
  useColorScheme,
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import Colors from "@/constants/colors";

export default function SettingsScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const insets = useSafeAreaInsets();
  const isWeb = Platform.OS === "web";
  const topPad = isWeb ? 67 : insets.top;

  const [notifBids, setNotifBids] = useState(true);
  const [notifMessages, setNotifMessages] = useState(true);
  const [notifSales, setNotifSales] = useState(true);

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <View
        style={[
          styles.header,
          {
            paddingTop: topPad + 12,
            backgroundColor: colors.backgroundSecondary,
            borderBottomColor: colors.border,
          },
        ]}
      >
        <Pressable
          onPress={() => router.back()}
          style={({ pressed }) => [styles.backBtn, pressed && { opacity: 0.7 }]}
        >
          <Feather name="arrow-left" size={22} color={colors.text} />
        </Pressable>
        <Text style={[styles.headerTitle, { color: colors.text }]}>Settings</Text>
      </View>

      <ScrollView
        contentInsetAdjustmentBehavior="automatic"
        showsVerticalScrollIndicator={false}
        contentContainerStyle={styles.content}
      >
        <Text style={[styles.sectionLabel, { color: colors.textTertiary }]}>
          NOTIFICATIONS
        </Text>
        <View style={[styles.card, { backgroundColor: colors.backgroundSecondary, borderColor: colors.border }]}>
          <ToggleRow
            label="Bid alerts"
            hint="Get notified when outbid"
            value={notifBids}
            onValueChange={setNotifBids}
          />
          <View style={[styles.divider, { backgroundColor: colors.border }]} />
          <ToggleRow
            label="New messages"
            hint="Chat notifications"
            value={notifMessages}
            onValueChange={setNotifMessages}
          />
          <View style={[styles.divider, { backgroundColor: colors.border }]} />
          <ToggleRow
            label="Sales activity"
            hint="Views and listing interest"
            value={notifSales}
            onValueChange={setNotifSales}
          />
        </View>

        <Text style={[styles.sectionLabel, { color: colors.textTertiary }]}>
          ACCOUNT
        </Text>
        <View style={[styles.card, { backgroundColor: colors.backgroundSecondary, borderColor: colors.border }]}>
          {[
            { label: "Edit Profile", icon: "user" },
            { label: "Change Password", icon: "lock" },
            { label: "Linked Accounts", icon: "link" },
            { label: "Privacy Settings", icon: "shield" },
          ].map((item, index, arr) => (
            <React.Fragment key={item.label}>
              <Pressable
                style={({ pressed }) => [
                  styles.menuRow,
                  pressed && { backgroundColor: colors.backgroundTertiary },
                ]}
              >
                <Feather name={item.icon as any} size={18} color={colors.tint} />
                <Text style={[styles.menuLabel, { color: colors.text }]}>
                  {item.label}
                </Text>
                <Feather name="chevron-right" size={16} color={colors.textTertiary} />
              </Pressable>
              {index < arr.length - 1 && (
                <View style={[styles.divider, { backgroundColor: colors.border }]} />
              )}
            </React.Fragment>
          ))}
        </View>

        <Text style={[styles.sectionLabel, { color: colors.textTertiary }]}>
          APP
        </Text>
        <View style={[styles.card, { backgroundColor: colors.backgroundSecondary, borderColor: colors.border }]}>
          {[
            { label: "Language", icon: "globe", value: "English" },
            { label: "Currency", icon: "dollar-sign", value: "AED" },
          ].map((item, index) => (
            <React.Fragment key={item.label}>
              <Pressable style={styles.menuRow}>
                <Feather name={item.icon as any} size={18} color={colors.tint} />
                <Text style={[styles.menuLabel, { color: colors.text }]}>
                  {item.label}
                </Text>
                <Text style={[styles.menuValue, { color: colors.textTertiary }]}>
                  {item.value}
                </Text>
                <Feather name="chevron-right" size={16} color={colors.textTertiary} />
              </Pressable>
              {index === 0 && (
                <View style={[styles.divider, { backgroundColor: colors.border }]} />
              )}
            </React.Fragment>
          ))}
        </View>

        <Pressable style={[styles.dangerBtn, { borderColor: "#EF4444" }]}>
          <Text style={[styles.dangerText, { color: "#EF4444" }]}>Sign Out</Text>
        </Pressable>

        <Text style={[styles.version, { color: colors.textTertiary }]}>
          GeoCore v1.0.0
        </Text>

        <View style={{ height: isWeb ? 34 : 100 }} />
      </ScrollView>
    </View>
  );
}

function ToggleRow({
  label,
  hint,
  value,
  onValueChange,
}: {
  label: string;
  hint: string;
  value: boolean;
  onValueChange: (v: boolean) => void;
}) {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  return (
    <View style={styles.toggleRow}>
      <View>
        <Text style={[styles.toggleLabel, { color: colors.text }]}>{label}</Text>
        <Text style={[styles.toggleHint, { color: colors.textTertiary }]}>{hint}</Text>
      </View>
      <Switch
        value={value}
        onValueChange={onValueChange}
        trackColor={{ false: colors.border, true: colors.tint }}
        thumbColor="#fff"
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  header: {
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: 16,
    paddingBottom: 14,
    borderBottomWidth: 1,
    gap: 12,
  },
  backBtn: {
    width: 36,
    height: 36,
    alignItems: "center",
    justifyContent: "center",
  },
  headerTitle: {
    flex: 1,
    fontSize: 20,
    fontFamily: "Inter_700Bold",
  },
  content: {
    padding: 16,
    gap: 8,
  },
  sectionLabel: {
    fontSize: 11,
    fontFamily: "Inter_600SemiBold",
    letterSpacing: 0.8,
    marginTop: 12,
    marginBottom: 4,
    marginLeft: 4,
  },
  card: {
    borderRadius: 14,
    borderWidth: 1,
    overflow: "hidden",
  },
  toggleRow: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    padding: 14,
  },
  toggleLabel: {
    fontSize: 15,
    fontFamily: "Inter_500Medium",
  },
  toggleHint: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
    marginTop: 2,
  },
  menuRow: {
    flexDirection: "row",
    alignItems: "center",
    padding: 14,
    gap: 12,
  },
  menuLabel: {
    flex: 1,
    fontSize: 15,
    fontFamily: "Inter_500Medium",
  },
  menuValue: {
    fontSize: 14,
    fontFamily: "Inter_400Regular",
  },
  divider: {
    height: 1,
    marginLeft: 44,
  },
  dangerBtn: {
    borderWidth: 1,
    borderRadius: 14,
    paddingVertical: 14,
    alignItems: "center",
    marginTop: 16,
  },
  dangerText: {
    fontSize: 15,
    fontFamily: "Inter_600SemiBold",
  },
  version: {
    textAlign: "center",
    fontSize: 12,
    fontFamily: "Inter_400Regular",
    marginTop: 8,
  },
});
