import { Feather } from "@expo/vector-icons";
import { router } from "expo-router";
import React from "react";
import {
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
  useColorScheme,
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import Colors from "@/constants/colors";

const NOTIFICATIONS = [
  {
    id: "n1",
    type: "bid",
    title: "You've been outbid!",
    body: "Someone placed a higher bid on Vintage Rolex Submariner 1968",
    time: "2 min ago",
    read: false,
  },
  {
    id: "n2",
    type: "message",
    title: "New message from TechDeals EG",
    body: "Sure, the iPhone is still available. Let me know when works for you.",
    time: "1h ago",
    read: false,
  },
  {
    id: "n3",
    type: "sale",
    title: "Your listing was viewed 50 times",
    body: "iPhone 15 Pro Max is getting traction. Consider featuring it!",
    time: "3h ago",
    read: true,
  },
  {
    id: "n4",
    type: "bid",
    title: "Auction ending soon",
    body: "DJI Mavic 3 Pro auction ends in less than 24 hours",
    time: "5h ago",
    read: true,
  },
  {
    id: "n5",
    type: "system",
    title: "Welcome to GeoCore!",
    body: "Your account is verified. Start selling or find great deals.",
    time: "Yesterday",
    read: true,
  },
];

const ICONS: Record<string, string> = {
  bid: "zap",
  message: "message-circle",
  sale: "trending-up",
  system: "bell",
};

const ICON_COLORS: Record<string, string> = {
  bid: "#0071CE",
  message: "#3B82F6",
  sale: "#10B981",
  system: "#6B7280",
};

export default function NotificationsScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const insets = useSafeAreaInsets();
  const isWeb = Platform.OS === "web";
  const topPad = isWeb ? 67 : insets.top;

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
        <Text style={[styles.headerTitle, { color: colors.text }]}>
          Notifications
        </Text>
        <Pressable style={({ pressed }) => [styles.backBtn, pressed && { opacity: 0.7 }]}>
          <Text style={[styles.markAll, { color: colors.tint }]}>Mark all read</Text>
        </Pressable>
      </View>

      <ScrollView
        contentInsetAdjustmentBehavior="automatic"
        showsVerticalScrollIndicator={false}
      >
        {NOTIFICATIONS.map((n, index) => (
          <Pressable
            key={n.id}
            style={({ pressed }) => [
              styles.notifRow,
              {
                backgroundColor: n.read
                  ? colors.backgroundSecondary
                  : colorScheme === "dark"
                  ? "#1A2A3A"
                  : "#EEF6FF",
                borderBottomColor: colors.border,
                opacity: pressed ? 0.85 : 1,
              },
            ]}
          >
            <View
              style={[
                styles.notifIcon,
                { backgroundColor: ICON_COLORS[n.type] + "20" },
              ]}
            >
              <Feather
                name={ICONS[n.type] as any}
                size={18}
                color={ICON_COLORS[n.type]}
              />
            </View>
            <View style={styles.notifContent}>
              <View style={styles.notifHeader}>
                <Text
                  style={[
                    styles.notifTitle,
                    {
                      color: colors.text,
                      fontFamily: n.read ? "Inter_500Medium" : "Inter_700Bold",
                    },
                  ]}
                >
                  {n.title}
                </Text>
                {!n.read && (
                  <View style={[styles.unreadDot, { backgroundColor: colors.tint }]} />
                )}
              </View>
              <Text style={[styles.notifBody, { color: colors.textSecondary }]}>
                {n.body}
              </Text>
              <Text style={[styles.notifTime, { color: colors.textTertiary }]}>
                {n.time}
              </Text>
            </View>
          </Pressable>
        ))}
        <View style={{ height: isWeb ? 34 : 100 }} />
      </ScrollView>
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
    gap: 8,
  },
  backBtn: {
    height: 36,
    alignItems: "center",
    justifyContent: "center",
    minWidth: 36,
  },
  headerTitle: {
    flex: 1,
    fontSize: 20,
    fontFamily: "Inter_700Bold",
  },
  markAll: {
    fontSize: 13,
    fontFamily: "Inter_500Medium",
  },
  notifRow: {
    flexDirection: "row",
    alignItems: "flex-start",
    padding: 16,
    gap: 12,
    borderBottomWidth: 1,
  },
  notifIcon: {
    width: 42,
    height: 42,
    borderRadius: 12,
    alignItems: "center",
    justifyContent: "center",
    flexShrink: 0,
  },
  notifContent: {
    flex: 1,
    gap: 4,
  },
  notifHeader: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
  },
  notifTitle: {
    fontSize: 14,
    flex: 1,
    lineHeight: 20,
  },
  unreadDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    marginLeft: 8,
  },
  notifBody: {
    fontSize: 13,
    fontFamily: "Inter_400Regular",
    lineHeight: 18,
  },
  notifTime: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
  },
});
