import { Feather } from "@expo/vector-icons";
import * as Haptics from "expo-haptics";
import { router } from "expo-router";
import React, { useEffect, useState } from "react";
import {
  Pressable,
  StyleSheet,
  Text,
  View,
  useColorScheme,
} from "react-native";

import Colors from "@/constants/colors";
import type { Listing } from "@/context/AppContext";
import { formatPrice } from "@/utils/format";

function useCountdown(endsAt?: string) {
  const getRemaining = () => {
    if (!endsAt) return null;
    const diff = new Date(endsAt).getTime() - Date.now();
    if (diff <= 0) return "Ended";
    const h = Math.floor(diff / 3600000);
    const m = Math.floor((diff % 3600000) / 60000);
    const s = Math.floor((diff % 60000) / 1000);
    if (h > 0) return `${h}h ${m}m`;
    if (m > 0) return `${m}m ${s}s`;
    return `${s}s`;
  };
  const [remaining, setRemaining] = useState(getRemaining());
  useEffect(() => {
    const id = setInterval(() => setRemaining(getRemaining()), 1000);
    return () => clearInterval(id);
  }, [endsAt]);
  return remaining;
}

export default function AuctionCard({ listing }: { listing: Listing }) {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const countdown = useCountdown(listing.auctionEndsAt);
  const currency = listing.currency ?? "AED";
  const isEndingSoon =
    listing.auctionEndsAt &&
    new Date(listing.auctionEndsAt).getTime() - Date.now() < 3600000;

  return (
    <Pressable
      onPress={() => {
        Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
        router.push({ pathname: "/listing/[id]", params: { id: listing.id } });
      }}
      style={({ pressed }) => [styles.card, pressed && { opacity: 0.93 }]}
    >
      <View style={[styles.imageArea, { backgroundColor: colors.backgroundTertiary }]}>
        <Feather name="image" size={28} color={colors.textTertiary} />

        <View style={styles.auctionBadge}>
          <Text style={styles.auctionBadgeText}>🔨 AUCTION</Text>
        </View>

        {countdown && (
          <View
            style={[
              styles.countdownBadge,
              { backgroundColor: isEndingSoon ? "#E53935" : "rgba(0,0,0,0.72)" },
            ]}
          >
            <Feather name="clock" size={10} color={colors.secondary} />
            <Text style={styles.countdownText}>{countdown}</Text>
          </View>
        )}
      </View>

      <View style={styles.info}>
        <Text numberOfLines={2} style={[styles.title, { color: colors.text }]}>
          {listing.title}
        </Text>
        <Text style={[styles.price, { color: colors.tint }]}>
          {formatPrice(listing.currentBid ?? 0, currency)}
        </Text>
        <Text style={[styles.bids, { color: colors.textTertiary }]}>
          {listing.bidCount ?? 0} bids
        </Text>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  card: {
    width: 160,
    borderRadius: 10,
    backgroundColor: "#fff",
    marginRight: 12,
    shadowColor: "#000",
    shadowOpacity: 0.1,
    shadowRadius: 8,
    shadowOffset: { width: 0, height: 2 },
    elevation: 3,
    overflow: "hidden",
  },
  imageArea: {
    height: 120,
    alignItems: "center",
    justifyContent: "center",
    position: "relative",
  },
  auctionBadge: {
    position: "absolute",
    top: 8,
    left: 8,
    backgroundColor: "#E53935",
    borderRadius: 4,
    paddingHorizontal: 6,
    paddingVertical: 2,
  },
  auctionBadgeText: {
    color: "#fff",
    fontSize: 9,
    fontFamily: "Inter_700Bold",
  },
  countdownBadge: {
    position: "absolute",
    top: 8,
    right: 8,
    borderRadius: 4,
    paddingHorizontal: 5,
    paddingVertical: 2,
    flexDirection: "row",
    alignItems: "center",
    gap: 3,
  },
  countdownText: {
    color: "#fff",
    fontSize: 9,
    fontFamily: "Inter_600SemiBold",
  },
  info: {
    padding: 10,
    gap: 2,
  },
  title: {
    fontSize: 12,
    fontFamily: "Inter_500Medium",
    lineHeight: 16,
  },
  price: {
    fontSize: 14,
    fontFamily: "Inter_700Bold",
    marginTop: 4,
  },
  bids: {
    fontSize: 11,
    fontFamily: "Inter_400Regular",
  },
});
