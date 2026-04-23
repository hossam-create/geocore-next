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
import type { Trip } from "@/types/crowdshipping";
import { crowdshippingAPI } from "@/utils/api";

export default function MyTripsScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];

  const [trips, setTrips] = useState<Trip[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const { data } = await crowdshippingAPI.listTrips({ mine: true });
      const payload = data?.data ?? data;
      setTrips(Array.isArray(payload) ? payload : []);
    } catch (err: any) {
      setError(err?.response?.data?.error ?? err?.message ?? "Failed to load trips");
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
        title="My Trips"
        right={
          <Pressable
            onPress={() => router.push("/crowdshipping/trips/new")}
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
          data={trips}
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
              onPress={() => router.push(`/crowdshipping/trips/${item.id}` as never)}
              style={({ pressed }) => [
                styles.row,
                { backgroundColor: colors.backgroundSecondary, borderColor: colors.border },
                pressed && { opacity: 0.85 },
              ]}
            >
              <View style={styles.rowHeader}>
                <Feather name="map-pin" size={14} color={colors.tint} />
                <Text style={[styles.route, { color: colors.text }]} numberOfLines={1}>
                  {item.origin_city} → {item.dest_city}
                </Text>
                <StatusPill status={item.status} colors={colors} />
              </View>
              <Text style={[styles.metaText, { color: colors.textTertiary }]}>
                Departs {new Date(item.departure_date).toLocaleDateString()} · Arrives{" "}
                {new Date(item.arrival_date).toLocaleDateString()}
              </Text>
            </Pressable>
          )}
          ListEmptyComponent={
            <View style={styles.empty}>
              <Feather name="compass" size={40} color={colors.textTertiary} />
              <Text style={[styles.emptyTitle, { color: colors.text }]}>
                You haven't posted any trips yet
              </Text>
              <Text style={[styles.emptyText, { color: colors.textTertiary }]}>
                {error ?? "Post a trip to start earning."}
              </Text>
              <Pressable
                onPress={() => router.push("/crowdshipping/trips/new")}
                style={[styles.cta, { backgroundColor: colors.tint }]}
              >
                <Text style={styles.ctaText}>Post a Trip</Text>
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
    active: "#10B981",
    matched: "#0071CE",
    in_transit: "#F59E0B",
    completed: "#6366F1",
    cancelled: colors.textTertiary,
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
  row: {
    padding: 14,
    borderRadius: 12,
    borderWidth: 1,
    gap: 6,
  },
  rowHeader: { flexDirection: "row", alignItems: "center", gap: 8 },
  route: { flex: 1, fontSize: 15, fontFamily: "Inter_600SemiBold" },
  metaText: { fontSize: 12, fontFamily: "Inter_400Regular" },
  pill: {
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: 10,
  },
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
