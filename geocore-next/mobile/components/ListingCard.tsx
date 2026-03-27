import { Feather, Ionicons } from "@expo/vector-icons";
import * as Haptics from "expo-haptics";
import { router } from "expo-router";
import React from "react";
import {
  Pressable,
  StyleSheet,
  Text,
  View,
  useColorScheme,
} from "react-native";
import Animated, {
  useAnimatedStyle,
  useSharedValue,
  withSpring,
} from "react-native-reanimated";

import Colors from "@/constants/colors";
import type { Listing } from "@/context/AppContext";
import { useAppContext } from "@/context/AppContext";
import { formatPrice, getAuctionTimeLeft } from "@/utils/format";

interface ListingCardProps {
  listing: Listing;
  compact?: boolean;
}

function HeartButton({ listing }: { listing: Listing }) {
  const { favorites, toggleFavorite } = useAppContext();
  const isFav = favorites.includes(listing.id);
  const scale = useSharedValue(1);
  const animStyle = useAnimatedStyle(() => ({
    transform: [{ scale: scale.value }],
  }));
  const handlePress = () => {
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    scale.value = withSpring(1.3, { damping: 3 }, () => {
      scale.value = withSpring(1);
    });
    toggleFavorite(listing.id);
  };
  return (
    <Pressable onPress={handlePress} hitSlop={8} style={styles.heartBtn}>
      <Animated.View style={animStyle}>
        <Ionicons
          name={isFav ? "heart" : "heart-outline"}
          size={18}
          color={isFav ? "#E53935" : "rgba(0,0,0,0.35)"}
        />
      </Animated.View>
    </Pressable>
  );
}

export default function ListingCard({ listing, compact = false }: ListingCardProps) {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const currency = listing.currency ?? "AED";

  const timeLeft =
    listing.isAuction && listing.auctionEndsAt
      ? getAuctionTimeLeft(listing.auctionEndsAt)
      : null;

  const handlePress = () => {
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    router.push({ pathname: "/listing/[id]", params: { id: listing.id } });
  };

  if (compact) {
    return (
      <Pressable
        onPress={handlePress}
        style={({ pressed }) => [styles.compactCard, pressed && { opacity: 0.92 }]}
      >
        <View style={[styles.compactImage, { backgroundColor: colors.backgroundTertiary }]}>
          <Feather name="image" size={22} color={colors.textTertiary} />
          <View
            style={[
              styles.typeBadge,
              {
                backgroundColor: listing.isAuction
                  ? "#E53935"
                  : colors.tint,
              },
            ]}
          >
            <Text style={styles.typeBadgeText}>
              {listing.isAuction ? "🔨 AUCTION" : "⚡ BUY NOW"}
            </Text>
          </View>
        </View>
        <View style={styles.compactInfo}>
          <Text numberOfLines={2} style={[styles.compactTitle, { color: colors.text }]}>
            {listing.title}
          </Text>
          <Text style={[styles.compactPrice, { color: colors.tint }]}>
            {listing.isAuction
              ? formatPrice(listing.currentBid ?? 0, currency)
              : formatPrice(listing.price, currency)}
          </Text>
          {timeLeft && (
            <Text style={styles.compactCountdown}>⏰ {timeLeft}</Text>
          )}
          <Text style={[styles.compactLocation, { color: colors.textTertiary }]}>
            📍 {listing.location}
          </Text>
        </View>
      </Pressable>
    );
  }

  return (
    <Pressable
      onPress={handlePress}
      style={({ pressed }) => [
        styles.gridCard,
        { backgroundColor: colors.backgroundSecondary },
        pressed && { opacity: 0.92 },
      ]}
    >
      <View style={[styles.gridImage, { backgroundColor: colors.backgroundTertiary }]}>
        <Feather name="image" size={28} color={colors.textTertiary} />

        <View
          style={[
            styles.typeBadge,
            {
              backgroundColor: listing.isAuction ? "#E53935" : colors.tint,
            },
          ]}
        >
          <Text style={styles.typeBadgeText}>
            {listing.isAuction ? "🔨 AUCTION" : "⚡ BUY NOW"}
          </Text>
        </View>

        {listing.isFeatured && (
          <View style={styles.featuredBadge}>
            <Text style={styles.featuredBadgeText}>⭐ TOP</Text>
          </View>
        )}

        <HeartButton listing={listing} />
      </View>

      <View style={styles.gridInfo}>
        <Text
          numberOfLines={2}
          style={[styles.gridTitle, { color: colors.text }]}
        >
          {listing.title}
        </Text>
        <Text style={[styles.gridPrice, { color: colors.tint }]}>
          {listing.isAuction
            ? formatPrice(listing.currentBid ?? 0, currency)
            : formatPrice(listing.price, currency)}
        </Text>
        {listing.isAuction && timeLeft && (
          <Text style={styles.gridCountdown}>⏰ {timeLeft}</Text>
        )}
        <Text style={[styles.gridLocation, { color: colors.textTertiary }]}>
          📍 {listing.location}
        </Text>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  compactCard: {
    width: 160,
    borderRadius: 10,
    backgroundColor: "#fff",
    marginRight: 12,
    shadowColor: "#000",
    shadowOpacity: 0.08,
    shadowRadius: 6,
    shadowOffset: { width: 0, height: 2 },
    elevation: 2,
    overflow: "hidden",
  },
  compactImage: {
    height: 110,
    alignItems: "center",
    justifyContent: "center",
    position: "relative",
  },
  compactInfo: {
    padding: 9,
    gap: 2,
  },
  compactTitle: {
    fontSize: 12,
    fontFamily: "Inter_500Medium",
    lineHeight: 16,
  },
  compactPrice: {
    fontSize: 13,
    fontFamily: "Inter_700Bold",
    marginTop: 3,
  },
  compactCountdown: {
    fontSize: 10,
    color: "#E53935",
    fontFamily: "Inter_500Medium",
  },
  compactLocation: {
    fontSize: 10,
    fontFamily: "Inter_400Regular",
    marginTop: 2,
  },
  gridCard: {
    flex: 1,
    borderRadius: 10,
    overflow: "hidden",
    shadowColor: "#000",
    shadowOpacity: 0.07,
    shadowRadius: 6,
    shadowOffset: { width: 0, height: 2 },
    elevation: 2,
    marginBottom: 10,
  },
  gridImage: {
    height: 150,
    alignItems: "center",
    justifyContent: "center",
    position: "relative",
  },
  gridInfo: {
    padding: 9,
    gap: 2,
  },
  gridTitle: {
    fontSize: 12,
    fontFamily: "Inter_500Medium",
    lineHeight: 16,
  },
  gridPrice: {
    fontSize: 15,
    fontFamily: "Inter_700Bold",
    marginTop: 3,
  },
  gridCountdown: {
    fontSize: 10,
    color: "#E53935",
    fontFamily: "Inter_500Medium",
  },
  gridLocation: {
    fontSize: 10,
    fontFamily: "Inter_400Regular",
    marginTop: 2,
  },
  typeBadge: {
    position: "absolute",
    top: 7,
    left: 7,
    borderRadius: 4,
    paddingHorizontal: 5,
    paddingVertical: 2,
  },
  typeBadgeText: {
    color: "#fff",
    fontSize: 8,
    fontFamily: "Inter_700Bold",
  },
  featuredBadge: {
    position: "absolute",
    top: 7,
    right: 7,
    backgroundColor: "#FFC220",
    borderRadius: 4,
    paddingHorizontal: 5,
    paddingVertical: 2,
  },
  featuredBadgeText: {
    color: "#1A1A1A",
    fontSize: 8,
    fontFamily: "Inter_700Bold",
  },
  heartBtn: {
    position: "absolute",
    bottom: 7,
    right: 7,
    backgroundColor: "rgba(255,255,255,0.9)",
    width: 28,
    height: 28,
    borderRadius: 14,
    alignItems: "center",
    justifyContent: "center",
  },
});
