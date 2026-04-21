import { Feather } from "@expo/vector-icons";
import { router, useLocalSearchParams } from "expo-router";
import React, { useCallback, useEffect, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
  useColorScheme,
} from "react-native";

import ScreenHeader from "@/components/ScreenHeader";
import Colors from "@/constants/colors";
import { useAuthStore } from "@/store/authStore";
import type { Trip } from "@/types/crowdshipping";
import { crowdshippingAPI } from "@/utils/api";

export default function TripDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const authUser = useAuthStore((s) => s.user);

  const [trip, setTrip] = useState<Trip | null>(null);
  const [loading, setLoading] = useState(true);
  const [cancelling, setCancelling] = useState(false);

  const load = useCallback(async () => {
    if (!id) return;
    try {
      const { data } = await crowdshippingAPI.getTrip(String(id));
      setTrip(data?.data ?? data);
    } catch {
      setTrip(null);
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    load();
  }, [load]);

  const onCancel = () => {
    if (!trip) return;
    Alert.alert("Cancel trip?", "Shoppers will no longer see this trip.", [
      { text: "Keep", style: "cancel" },
      {
        text: "Cancel trip",
        style: "destructive",
        onPress: async () => {
          setCancelling(true);
          try {
            await crowdshippingAPI.cancelTrip(trip.id);
            Alert.alert("Cancelled", "Your trip was cancelled.", [
              { text: "OK", onPress: () => router.back() },
            ]);
          } catch (err: any) {
            Alert.alert(
              "Error",
              err?.response?.data?.error ?? err?.message ?? "Failed to cancel"
            );
          } finally {
            setCancelling(false);
          }
        },
      },
    ]);
  };

  if (loading) {
    return (
      <View style={[styles.container, { backgroundColor: colors.background }]}>
        <ScreenHeader title="Trip" />
        <View style={styles.center}>
          <ActivityIndicator color={colors.tint} />
        </View>
      </View>
    );
  }

  if (!trip) {
    return (
      <View style={[styles.container, { backgroundColor: colors.background }]}>
        <ScreenHeader title="Trip" />
        <View style={styles.center}>
          <Feather name="alert-circle" size={32} color={colors.textTertiary} />
          <Text style={[styles.muted, { color: colors.textTertiary }]}>
            Trip not found
          </Text>
        </View>
      </View>
    );
  }

  const isOwner = authUser?.id === trip.traveler_id;

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <ScreenHeader title="Trip" />
      <ScrollView contentContainerStyle={styles.content} showsVerticalScrollIndicator={false}>
        <View
          style={[
            styles.card,
            { backgroundColor: colors.backgroundSecondary, borderColor: colors.border },
          ]}
        >
          <Row icon="send" label="From" value={`${trip.origin_city}, ${trip.origin_country}`} colors={colors} />
          <Row icon="map-pin" label="To" value={`${trip.dest_city}, ${trip.dest_country}`} colors={colors} />
          <Row
            icon="calendar"
            label="Departs"
            value={new Date(trip.departure_date).toLocaleString()}
            colors={colors}
          />
          <Row
            icon="calendar"
            label="Arrives"
            value={new Date(trip.arrival_date).toLocaleString()}
            colors={colors}
          />
          <Row
            icon="package"
            label="Capacity"
            value={`${trip.available_weight} kg · up to ${trip.max_items} items`}
            colors={colors}
          />
          {(trip.price_per_kg > 0 || trip.base_price > 0) && (
            <Row
              icon="tag"
              label="Price"
              value={`${trip.price_per_kg > 0 ? `${trip.price_per_kg} ${trip.currency}/kg` : ""}${
                trip.price_per_kg > 0 && trip.base_price > 0 ? " · " : ""
              }${trip.base_price > 0 ? `base ${trip.base_price} ${trip.currency}` : ""}`}
              colors={colors}
            />
          )}
          <Row icon="activity" label="Status" value={trip.status} colors={colors} />
          {trip.notes ? (
            <Row icon="file-text" label="Notes" value={trip.notes} colors={colors} />
          ) : null}
        </View>

        {isOwner && trip.status === "active" && (
          <Pressable
            onPress={onCancel}
            disabled={cancelling}
            style={({ pressed }) => [
              styles.dangerBtn,
              { borderColor: colors.error, opacity: pressed || cancelling ? 0.8 : 1 },
            ]}
          >
            {cancelling ? (
              <ActivityIndicator color={colors.error} />
            ) : (
              <Text style={[styles.dangerText, { color: colors.error }]}>Cancel trip</Text>
            )}
          </Pressable>
        )}
      </ScrollView>
    </View>
  );
}

function Row({
  icon,
  label,
  value,
  colors,
}: {
  icon: keyof typeof Feather.glyphMap;
  label: string;
  value: string;
  colors: (typeof Colors)["light"];
}) {
  return (
    <View style={styles.row}>
      <View style={[styles.rowIcon, { backgroundColor: colors.tint + "22" }]}>
        <Feather name={icon} size={14} color={colors.tint} />
      </View>
      <View style={{ flex: 1 }}>
        <Text style={[styles.rowLabel, { color: colors.textTertiary }]}>{label}</Text>
        <Text style={[styles.rowValue, { color: colors.text }]}>{value}</Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: 8 },
  muted: { fontSize: 14, fontFamily: "Inter_400Regular" },
  content: { padding: 16, gap: 14 },
  card: {
    borderRadius: 14,
    borderWidth: 1,
    padding: 6,
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: 10,
    paddingVertical: 10,
    gap: 12,
  },
  rowIcon: {
    width: 30,
    height: 30,
    borderRadius: 8,
    alignItems: "center",
    justifyContent: "center",
  },
  rowLabel: { fontSize: 11, fontFamily: "Inter_500Medium", textTransform: "uppercase" },
  rowValue: { fontSize: 14, fontFamily: "Inter_500Medium", marginTop: 2 },
  dangerBtn: {
    marginTop: 4,
    borderWidth: 1,
    borderRadius: 12,
    paddingVertical: 13,
    alignItems: "center",
  },
  dangerText: { fontSize: 15, fontFamily: "Inter_600SemiBold" },
});
