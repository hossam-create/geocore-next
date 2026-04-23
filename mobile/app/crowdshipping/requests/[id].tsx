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
import type { DeliveryRequest, MatchResult } from "@/types/crowdshipping";
import { crowdshippingAPI } from "@/utils/api";

export default function DeliveryRequestDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const authUser = useAuthStore((s) => s.user);

  const [request, setRequest] = useState<DeliveryRequest | null>(null);
  const [matches, setMatches] = useState<MatchResult[]>([]);
  const [loading, setLoading] = useState(true);
  const [finding, setFinding] = useState(false);
  const [acting, setActing] = useState(false);

  const load = useCallback(async () => {
    if (!id) return;
    try {
      const { data } = await crowdshippingAPI.getDeliveryRequest(String(id));
      setRequest(data?.data ?? data);
    } catch {
      setRequest(null);
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    load();
  }, [load]);

  const findTravelers = async () => {
    if (!request) return;
    setFinding(true);
    try {
      const { data } = await crowdshippingAPI.findTravelers(request.id);
      const payload = data?.data ?? data;
      const results: MatchResult[] = Array.isArray(payload?.matches)
        ? payload.matches
        : Array.isArray(payload)
          ? payload
          : [];
      setMatches(results);
      if (results.length === 0) {
        Alert.alert("No matches yet", "No travelers match this route right now. Try again later.");
      }
    } catch (err: any) {
      Alert.alert("Error", err?.response?.data?.error ?? err?.message ?? "Failed to find travelers");
    } finally {
      setFinding(false);
    }
  };

  const acceptMatch = async (tripId: string) => {
    if (!request) return;
    setActing(true);
    try {
      await crowdshippingAPI.acceptMatch(request.id, { trip_id: tripId });
      Alert.alert("Accepted", "The traveler has been notified.");
      await load();
    } catch (err: any) {
      Alert.alert("Error", err?.response?.data?.error ?? err?.message ?? "Failed to accept");
    } finally {
      setActing(false);
    }
  };

  const confirmDelivery = async () => {
    if (!request) return;
    Alert.alert("Confirm delivery?", "Only do this once you've received the item.", [
      { text: "Cancel", style: "cancel" },
      {
        text: "Confirm",
        onPress: async () => {
          setActing(true);
          try {
            await crowdshippingAPI.confirmDelivery(request.id);
            Alert.alert("Confirmed", "The traveler will be paid the reward.");
            await load();
          } catch (err: any) {
            Alert.alert(
              "Error",
              err?.response?.data?.error ?? err?.message ?? "Failed to confirm"
            );
          } finally {
            setActing(false);
          }
        },
      },
    ]);
  };

  if (loading) {
    return (
      <View style={[styles.container, { backgroundColor: colors.background }]}>
        <ScreenHeader title="Request" />
        <View style={styles.center}>
          <ActivityIndicator color={colors.tint} />
        </View>
      </View>
    );
  }

  if (!request) {
    return (
      <View style={[styles.container, { backgroundColor: colors.background }]}>
        <ScreenHeader title="Request" />
        <View style={styles.center}>
          <Feather name="alert-circle" size={32} color={colors.textTertiary} />
          <Text style={[styles.muted, { color: colors.textTertiary }]}>Request not found</Text>
        </View>
      </View>
    );
  }

  const isBuyer = authUser?.id === request.buyer_id;

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <ScreenHeader title="Request" />
      <ScrollView contentContainerStyle={styles.content} showsVerticalScrollIndicator={false}>
        <View
          style={[
            styles.card,
            { backgroundColor: colors.backgroundSecondary, borderColor: colors.border },
          ]}
        >
          <Row icon="package" label="Item" value={request.item_name} colors={colors} />
          {request.item_description ? (
            <Row icon="file-text" label="Description" value={request.item_description} colors={colors} />
          ) : null}
          <Row
            icon="dollar-sign"
            label="Item price"
            value={`${request.item_price} ${request.currency}`}
            colors={colors}
          />
          {request.item_weight ? (
            <Row icon="activity" label="Weight" value={`${request.item_weight} kg`} colors={colors} />
          ) : null}
          <Row
            icon="map-pin"
            label="Pickup"
            value={`${request.pickup_city}, ${request.pickup_country}`}
            colors={colors}
          />
          <Row
            icon="navigation"
            label="Delivery"
            value={`${request.delivery_city}, ${request.delivery_country}`}
            colors={colors}
          />
          <Row
            icon="gift"
            label="Reward"
            value={`${request.reward} ${request.currency}`}
            colors={colors}
          />
          <Row icon="activity" label="Status" value={request.status} colors={colors} />
          {request.notes ? (
            <Row icon="edit-3" label="Notes" value={request.notes} colors={colors} />
          ) : null}
        </View>

        {isBuyer && request.status === "pending" && (
          <Pressable
            onPress={findTravelers}
            disabled={finding}
            style={({ pressed }) => [
              styles.primaryBtn,
              { backgroundColor: colors.tint, opacity: pressed || finding ? 0.85 : 1 },
            ]}
          >
            {finding ? (
              <ActivityIndicator color="#fff" />
            ) : (
              <Text style={styles.primaryText}>Find travelers</Text>
            )}
          </Pressable>
        )}

        {isBuyer &&
          (request.status === "in_transit" ||
            request.status === "picked_up" ||
            request.status === "accepted") && (
            <Pressable
              onPress={confirmDelivery}
              disabled={acting}
              style={({ pressed }) => [
                styles.primaryBtn,
                { backgroundColor: colors.success, opacity: pressed || acting ? 0.85 : 1 },
              ]}
            >
              {acting ? (
                <ActivityIndicator color="#fff" />
              ) : (
                <Text style={styles.primaryText}>Confirm delivery received</Text>
              )}
            </Pressable>
          )}

        {matches.length > 0 && (
          <View style={styles.matchesSection}>
            <Text style={[styles.sectionTitle, { color: colors.text }]}>
              Matching travelers
            </Text>
            {matches.map((m) => (
              <View
                key={m.trip_id}
                style={[
                  styles.matchRow,
                  { backgroundColor: colors.backgroundSecondary, borderColor: colors.border },
                ]}
              >
                <View style={{ flex: 1 }}>
                  <Text style={[styles.matchRoute, { color: colors.text }]} numberOfLines={1}>
                    {m.origin_city ?? "—"} → {m.dest_city ?? "—"}
                  </Text>
                  <Text style={[styles.matchMeta, { color: colors.textTertiary }]}>
                    {m.departure_date
                      ? new Date(m.departure_date).toLocaleDateString()
                      : ""}
                    {m.price_per_kg
                      ? ` · ${m.price_per_kg} ${m.currency ?? ""}/kg`
                      : ""}
                    {m.score ? ` · score ${Math.round(m.score)}` : ""}
                  </Text>
                </View>
                <Pressable
                  onPress={() => router.push(`/crowdshipping/trips/${m.trip_id}` as never)}
                  style={({ pressed }) => [
                    styles.outlineBtn,
                    { borderColor: colors.tint, opacity: pressed ? 0.8 : 1 },
                  ]}
                >
                  <Text style={[styles.outlineText, { color: colors.tint }]}>View</Text>
                </Pressable>
                {isBuyer && request.status === "pending" && (
                  <Pressable
                    onPress={() => acceptMatch(m.trip_id)}
                    disabled={acting}
                    style={({ pressed }) => [
                      styles.primarySmallBtn,
                      { backgroundColor: colors.tint, opacity: pressed || acting ? 0.85 : 1 },
                    ]}
                  >
                    <Text style={styles.primarySmallText}>Accept</Text>
                  </Pressable>
                )}
              </View>
            ))}
          </View>
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
  rowLabel: {
    fontSize: 11,
    fontFamily: "Inter_500Medium",
    textTransform: "uppercase",
  },
  rowValue: { fontSize: 14, fontFamily: "Inter_500Medium", marginTop: 2 },
  primaryBtn: {
    paddingVertical: 13,
    borderRadius: 12,
    alignItems: "center",
  },
  primaryText: { color: "#fff", fontSize: 15, fontFamily: "Inter_600SemiBold" },
  matchesSection: { marginTop: 8, gap: 10 },
  sectionTitle: { fontSize: 16, fontFamily: "Inter_600SemiBold", marginBottom: 4 },
  matchRow: {
    flexDirection: "row",
    alignItems: "center",
    borderWidth: 1,
    borderRadius: 12,
    padding: 12,
    gap: 8,
  },
  matchRoute: { fontSize: 14, fontFamily: "Inter_600SemiBold" },
  matchMeta: { fontSize: 12, fontFamily: "Inter_400Regular", marginTop: 2 },
  outlineBtn: {
    borderWidth: 1,
    paddingHorizontal: 12,
    paddingVertical: 7,
    borderRadius: 8,
  },
  outlineText: { fontSize: 12, fontFamily: "Inter_600SemiBold" },
  primarySmallBtn: {
    paddingHorizontal: 12,
    paddingVertical: 7,
    borderRadius: 8,
  },
  primarySmallText: {
    color: "#fff",
    fontSize: 12,
    fontFamily: "Inter_600SemiBold",
  },
});
