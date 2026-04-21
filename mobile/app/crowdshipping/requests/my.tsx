import { Feather } from "@expo/vector-icons";
import { router } from "expo-router";
import React, { useCallback, useEffect, useState } from "react";
import {
  ActivityIndicator,
  FlatList,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  View,
  useColorScheme,
} from "react-native";

import ScreenHeader from "@/components/ScreenHeader";
import Colors from "@/constants/colors";
import type { DeliveryRequest } from "@/types/crowdshipping";
import { crowdshippingAPI } from "@/utils/api";

export default function MyRequestsScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];

  const [requests, setRequests] = useState<DeliveryRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const { data } = await crowdshippingAPI.listDeliveryRequests({ mine: true });
      const payload = data?.data ?? data;
      setRequests(Array.isArray(payload) ? payload : []);
    } catch (err: any) {
      setError(err?.response?.data?.error ?? err?.message ?? "Failed to load requests");
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <ScreenHeader
        title="My Requests"
        right={
          <Pressable
            onPress={() => router.push("/crowdshipping/requests/new")}
            style={({ pressed }) => [
              styles.addBtn,
              { backgroundColor: colors.tint, opacity: pressed ? 0.8 : 1 },
            ]}
          >
            <Feather name="plus" size={18} color="#fff" />
          </Pressable>
        }
      />

      {loading ? (
        <View style={styles.center}>
          <ActivityIndicator color={colors.tint} />
        </View>
      ) : (
        <FlatList
          data={requests}
          keyExtractor={(item) => item.id}
          contentContainerStyle={styles.list}
          refreshControl={
            <RefreshControl
              refreshing={refreshing}
              onRefresh={() => {
                setRefreshing(true);
                load();
              }}
              tintColor={colors.tint}
            />
          }
          renderItem={({ item }) => (
            <Pressable
              onPress={() => router.push(`/crowdshipping/requests/${item.id}` as never)}
              style={({ pressed }) => [
                styles.row,
                { backgroundColor: colors.backgroundSecondary, borderColor: colors.border },
                pressed && { opacity: 0.85 },
              ]}
            >
              <View style={styles.rowHeader}>
                <Feather name="package" size={14} color={colors.tint} />
                <Text style={[styles.title, { color: colors.text }]} numberOfLines={1}>
                  {item.item_name}
                </Text>
                <StatusPill status={item.status} colors={colors} />
              </View>
              <Text style={[styles.metaText, { color: colors.textTertiary }]} numberOfLines={1}>
                {item.pickup_city} → {item.delivery_city} · {item.reward} {item.currency}
              </Text>
            </Pressable>
          )}
          ListEmptyComponent={
            <View style={styles.empty}>
              <Feather name="inbox" size={40} color={colors.textTertiary} />
              <Text style={[styles.emptyTitle, { color: colors.text }]}>
                No requests yet
              </Text>
              <Text style={[styles.emptyText, { color: colors.textTertiary }]}>
                {error ?? "Post a request to have travelers bring items for you."}
              </Text>
              <Pressable
                onPress={() => router.push("/crowdshipping/requests/new")}
                style={[styles.cta, { backgroundColor: colors.tint }]}
              >
                <Text style={styles.ctaText}>Post a Request</Text>
              </Pressable>
            </View>
          }
        />
      )}
    </View>
  );
}

function StatusPill({
  status,
  colors,
}: {
  status: string;
  colors: (typeof Colors)["light"];
}) {
  const palette: Record<string, string> = {
    pending: "#F59E0B",
    matched: "#0071CE",
    accepted: "#10B981",
    locked: "#6366F1",
    picked_up: "#8B5CF6",
    in_transit: "#F59E0B",
    delivered: "#10B981",
    cancelled: colors.textTertiary,
    disputed: colors.error,
  };
  const color = palette[status] ?? colors.textTertiary;
  return (
    <View style={[styles.pill, { backgroundColor: color + "22" }]}>
      <Text style={[styles.pillText, { color }]}>{status.replace("_", " ")}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  center: { flex: 1, alignItems: "center", justifyContent: "center" },
  addBtn: {
    width: 36,
    height: 36,
    borderRadius: 18,
    alignItems: "center",
    justifyContent: "center",
  },
  list: { padding: 16, gap: 10 },
  row: { padding: 14, borderRadius: 12, borderWidth: 1, gap: 6 },
  rowHeader: { flexDirection: "row", alignItems: "center", gap: 8 },
  title: { flex: 1, fontSize: 15, fontFamily: "Inter_600SemiBold" },
  metaText: { fontSize: 12, fontFamily: "Inter_400Regular" },
  pill: { paddingHorizontal: 8, paddingVertical: 2, borderRadius: 10 },
  pillText: {
    fontSize: 10,
    fontFamily: "Inter_600SemiBold",
    textTransform: "capitalize",
  },
  empty: {
    alignItems: "center",
    paddingTop: 80,
    gap: 10,
    paddingHorizontal: 40,
  },
  emptyTitle: { fontSize: 18, fontFamily: "Inter_600SemiBold" },
  emptyText: { fontSize: 14, fontFamily: "Inter_400Regular", textAlign: "center" },
  cta: {
    marginTop: 8,
    paddingHorizontal: 24,
    paddingVertical: 12,
    borderRadius: 12,
  },
  ctaText: {
    color: "#fff",
    fontSize: 15,
    fontFamily: "Inter_600SemiBold",
  },
});
