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

export default function BrowseTripsScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];

  const [trips, setTrips] = useState<Trip[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const { data } = await crowdshippingAPI.listTrips();
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
        title="Browse Trips"
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
          renderItem={({ item }) => <TripRow trip={item} colors={colors} />}
          ListEmptyComponent={
            <View style={styles.empty}>
              <Feather name="map" size={40} color={colors.textTertiary} />
              <Text style={[styles.emptyTitle, { color: colors.text }]}>
                No active trips yet
              </Text>
              <Text style={[styles.emptyText, { color: colors.textTertiary }]}>
                {error ?? "Check back soon, or post a trip of your own."}
              </Text>
            </View>
          }
        />
      )}
    </View>
  );
}

function TripRow({ trip, colors }: { trip: Trip; colors: (typeof Colors)["light"] }) {
  const dep = new Date(trip.departure_date);
  const arr = new Date(trip.arrival_date);
  return (
    <Pressable
      onPress={() => router.push(`/crowdshipping/trips/${trip.id}` as never)}
      style={({ pressed }) => [
        styles.row,
        { backgroundColor: colors.backgroundSecondary, borderColor: colors.border },
        pressed && { opacity: 0.85 },
      ]}
    >
      <View style={styles.rowHeader}>
        <Feather name="send" size={16} color={colors.tint} />
        <Text style={[styles.route, { color: colors.text }]} numberOfLines={1}>
          {trip.origin_city}, {trip.origin_country} → {trip.dest_city}, {trip.dest_country}
        </Text>
      </View>
      <View style={styles.rowMeta}>
        <Feather name="calendar" size={12} color={colors.textTertiary} />
        <Text style={[styles.metaText, { color: colors.textTertiary }]}>
          {dep.toLocaleDateString()} – {arr.toLocaleDateString()}
        </Text>
      </View>
      <View style={styles.rowMeta}>
        <Feather name="package" size={12} color={colors.textTertiary} />
        <Text style={[styles.metaText, { color: colors.textTertiary }]}>
          {trip.available_weight > 0 ? `${trip.available_weight} kg available` : "Weight not specified"}
          {trip.price_per_kg > 0 && ` · ${trip.price_per_kg} ${trip.currency}/kg`}
        </Text>
      </View>
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
  row: {
    padding: 14,
    borderRadius: 12,
    borderWidth: 1,
    gap: 6,
  },
  rowHeader: { flexDirection: "row", alignItems: "center", gap: 8 },
  route: { flex: 1, fontSize: 15, fontFamily: "Inter_600SemiBold" },
  rowMeta: { flexDirection: "row", alignItems: "center", gap: 6 },
  metaText: { fontSize: 12, fontFamily: "Inter_400Regular" },
  empty: {
    alignItems: "center",
    paddingTop: 80,
    gap: 10,
    paddingHorizontal: 40,
  },
  emptyTitle: { fontSize: 18, fontFamily: "Inter_600SemiBold" },
  emptyText: { fontSize: 14, fontFamily: "Inter_400Regular", textAlign: "center" },
});
