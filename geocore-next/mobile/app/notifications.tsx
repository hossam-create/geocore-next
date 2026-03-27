import { Feather } from "@expo/vector-icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { router } from "expo-router";
import React from "react";
import {
  ActivityIndicator,
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
import { notificationsAPI } from "@/utils/api";

const ICON_MAP: Record<string, string> = {
  new_bid: "zap",
  outbid: "alert-triangle",
  auction_won: "award",
  auction_ended: "clock",
  new_message: "message-circle",
  listing_approved: "check-circle",
  listing_rejected: "x-circle",
  payment_success: "credit-card",
  payment_failed: "alert-circle",
  escrow_released: "unlock",
  new_review: "star",
};

const COLOR_MAP: Record<string, string> = {
  new_bid: "#0071CE",
  outbid: "#EF4444",
  auction_won: "#10B981",
  auction_ended: "#6B7280",
  new_message: "#3B82F6",
  listing_approved: "#10B981",
  listing_rejected: "#EF4444",
  payment_success: "#10B981",
  payment_failed: "#EF4444",
  escrow_released: "#8B5CF6",
  new_review: "#F59E0B",
};

function formatTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  if (diffMins < 1) return "Just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  const diffHrs = Math.floor(diffMins / 60);
  if (diffHrs < 24) return `${diffHrs}h ago`;
  const diffDays = Math.floor(diffHrs / 24);
  if (diffDays === 1) return "Yesterday";
  return `${diffDays}d ago`;
}

export default function NotificationsScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const insets = useSafeAreaInsets();
  const isWeb = Platform.OS === "web";
  const topPad = isWeb ? 67 : insets.top;
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ["notifications"],
    queryFn: () => notificationsAPI.list().then((r) => r.data.data ?? []),
    retry: false,
  });

  const markAllRead = useMutation({
    mutationFn: () => notificationsAPI.markAllRead(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });

  const markOneRead = useMutation({
    mutationFn: (id: string) => notificationsAPI.markRead(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });

  const notifications: any[] = data ?? [];

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
        <Pressable
          onPress={() => markAllRead.mutate()}
          style={({ pressed }) => [styles.backBtn, pressed && { opacity: 0.7 }]}
        >
          <Text style={[styles.markAll, { color: colors.tint }]}>Mark all read</Text>
        </Pressable>
      </View>

      {isLoading ? (
        <View style={styles.loading}>
          <ActivityIndicator size="large" color={colors.tint} />
        </View>
      ) : notifications.length === 0 ? (
        <View style={styles.empty}>
          <Feather name="bell-off" size={48} color={colors.textTertiary} />
          <Text style={[styles.emptyText, { color: colors.textSecondary }]}>
            No notifications yet
          </Text>
        </View>
      ) : (
        <ScrollView
          contentInsetAdjustmentBehavior="automatic"
          showsVerticalScrollIndicator={false}
        >
          {notifications.map((n: any) => {
            const iconName = ICON_MAP[n.type] ?? "bell";
            const iconColor = COLOR_MAP[n.type] ?? "#6B7280";
            return (
              <Pressable
                key={n.id}
                onPress={() => {
                  if (!n.read) markOneRead.mutate(n.id);
                }}
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
                    { backgroundColor: iconColor + "20" },
                  ]}
                >
                  <Feather name={iconName as any} size={18} color={iconColor} />
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
                    {formatTime(n.created_at)}
                  </Text>
                </View>
              </Pressable>
            );
          })}
          <View style={{ height: isWeb ? 34 : 100 }} />
        </ScrollView>
      )}
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
  loading: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
  },
  empty: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: 16,
  },
  emptyText: {
    fontSize: 16,
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
