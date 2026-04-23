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

export default function BrowseRequestsScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];

  const [requests, setRequests] = useState<DeliveryRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const { data } = await crowdshippingAPI.listDeliveryRequests({ status: "pending" });
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
        title="Delivery Requests"
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
          renderItem={({ item }) => <RequestRow item={item} colors={colors} />}
          ListEmptyComponent={
            <View style={styles.empty}>
              <Feather name="shopping-bag" size={40} color={colors.textTertiary} />
              <Text style={[styles.emptyTitle, { color: colors.text }]}>
                No open requests
              </Text>
              <Text style={[styles.emptyText, { color: colors.textTertiary }]}>
                {error ?? "Check back soon."}
              </Text>
            </View>
          }
        />
      )}
    </View>
  );
}

function RequestRow({
  item,
  colors,
}: {
  item: DeliveryRequest;
  colors: (typeof Colors)["light"];
}) {
  return (
    <Pressable
      onPress={() => router.push(`/crowdshipping/requests/${item.id}` as never)}
      style={({ pressed }) => [
        styles.row,
        { backgroundColor: colors.backgroundSecondary, borderColor: colors.border },
        pressed && { opacity: 0.85 },
      ]}
    >
      <View style={styles.rowHeader}>
        <Feather name="package" size={16} color={colors.tint} />
        <Text style={[styles.title, { color: colors.text }]} numberOfLines={1}>
          {item.item_name}
        </Text>
        <Text style={[styles.reward, { color: colors.success }]}>
          +{item.reward} {item.currency}
        </Text>
      </View>
      <View style={styles.rowMeta}>
        <Feather name="map" size={12} color={colors.textTertiary} />
        <Text style={[styles.metaText, { color: colors.textTertiary }]} numberOfLines={1}>
          {item.pickup_city}, {item.pickup_country} → {item.delivery_city}, {item.delivery_country}
        </Text>
      </View>
      {item.item_weight ? (
        <View style={styles.rowMeta}>
          <Feather name="activity" size={12} color={colors.textTertiary} />
          <Text style={[styles.metaText, { color: colors.textTertiary }]}>
            {item.item_weight} kg
          </Text>
        </View>
      ) : null}
    </Pressable>
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
  reward: { fontSize: 14, fontFamily: "Inter_700Bold" },
  rowMeta: { flexDirection: "row", alignItems: "center", gap: 6 },
  metaText: { fontSize: 12, fontFamily: "Inter_400Regular", flex: 1 },
  empty: {
    alignItems: "center",
    paddingTop: 80,
    gap: 10,
    paddingHorizontal: 40,
  },
  emptyTitle: { fontSize: 18, fontFamily: "Inter_600SemiBold" },
  emptyText: { fontSize: 14, fontFamily: "Inter_400Regular", textAlign: "center" },
});
