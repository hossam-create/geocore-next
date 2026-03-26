import { Feather, Ionicons } from "@expo/vector-icons";
import * as Haptics from "expo-haptics";
import { router, useLocalSearchParams } from "expo-router";
import React, { useState } from "react";
import {
  Alert,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Switch,
  Text,
  TextInput,
  View,
  useColorScheme,
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import Colors from "@/constants/colors";
import { useAppContext } from "@/context/AppContext";
import {
  formatAuctionType,
  formatPrice,
  formatRelativeTime,
  getAuctionTimeLeft,
  getConditionColor,
  getConditionLabel,
} from "@/utils/format";

export default function ListingDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const insets = useSafeAreaInsets();
  const isWeb = Platform.OS === "web";
  const { listings, favorites, toggleFavorite, startConversation, user } =
    useAppContext();

  const listing = listings.find((l) => l.id === id);
  const isFav = favorites.includes(id ?? "");
  const [bidAmount, setBidAmount] = useState("");
  const [autoBid, setAutoBid] = useState(false);
  const [autoBidMax, setAutoBidMax] = useState("");

  if (!listing) {
    return (
      <View style={[styles.notFound, { backgroundColor: colors.background }]}>
        <Text style={[styles.notFoundText, { color: colors.text }]}>
          Listing not found
        </Text>
        <Pressable onPress={() => router.back()}>
          <Text style={[styles.backLink, { color: colors.tint }]}>Go back</Text>
        </Pressable>
      </View>
    );
  }

  const currency = listing.currency ?? "AED";

  const timeLeft =
    listing.isAuction && listing.auctionEndsAt
      ? getAuctionTimeLeft(listing.auctionEndsAt)
      : null;

  const isEndingSoon = listing.auctionEndsAt
    ? new Date(listing.auctionEndsAt).getTime() - Date.now() < 5 * 60 * 1000
    : false;

  const minNextBid =
    listing.auctionType === "reverse"
      ? (listing.currentBid ?? 0) + 50
      : (listing.currentBid ?? 0) + 100;

  const handleContact = () => {
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
    const convId = startConversation(listing);
    router.push({ pathname: "/conversation/[id]", params: { id: convId } });
  };

  const handleFavorite = () => {
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    toggleFavorite(listing.id);
  };

  const handleBid = () => {
    const amount = bidAmount ? parseFloat(bidAmount) : minNextBid;
    if (amount < minNextBid) {
      Alert.alert(
        "Bid too low",
        `Minimum bid is ${formatPrice(minNextBid, currency)}`
      );
      return;
    }
    if (autoBid && autoBidMax) {
      const max = parseFloat(autoBidMax);
      if (max < amount) {
        Alert.alert("Invalid auto-bid", "Max auto-bid must be ≥ your bid.");
        return;
      }
    }
    Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
    const msg = autoBid && autoBidMax
      ? `Bid of ${formatPrice(amount, currency)} placed.\nAuto-bid active up to ${formatPrice(parseFloat(autoBidMax), currency)}.`
      : `Your bid of ${formatPrice(amount, currency)} has been placed.`;
    Alert.alert("Bid Placed ⚡", msg, [{ text: "Great!" }]);
    setBidAmount("");
  };

  const handleReport = () => {
    Alert.alert(
      "Report Listing",
      "Why are you reporting this listing?",
      [
        { text: "Spam or scam", onPress: () => Alert.alert("Reported", "Thank you. We'll review this within 24 hours.") },
        { text: "Misleading description", onPress: () => Alert.alert("Reported", "Thank you. We'll review this within 24 hours.") },
        { text: "Prohibited item", onPress: () => Alert.alert("Reported", "Thank you. We'll review this within 24 hours.") },
        { text: "Cancel", style: "cancel" },
      ]
    );
  };

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <View style={styles.imageContainer}>
        <View style={[styles.imagePlaceholder, { backgroundColor: colors.backgroundTertiary }]}>
          <Feather name="image" size={48} color={colors.textTertiary} />
        </View>

        <View
          style={[
            styles.topBar,
            { paddingTop: isWeb ? 67 : insets.top + 8 },
          ]}
        >
          <Pressable
            onPress={() => router.back()}
            style={({ pressed }) => [
              styles.circleBtn,
              { backgroundColor: "rgba(0,0,0,0.4)", opacity: pressed ? 0.7 : 1 },
            ]}
          >
            <Feather name="arrow-left" size={20} color="#fff" />
          </Pressable>
          <View style={styles.topBarRight}>
            <Pressable
              onPress={handleFavorite}
              style={({ pressed }) => [
                styles.circleBtn,
                { backgroundColor: "rgba(0,0,0,0.4)", opacity: pressed ? 0.7 : 1 },
              ]}
            >
              <Ionicons
                name={isFav ? "heart" : "heart-outline"}
                size={20}
                color={isFav ? "#EF4444" : "#fff"}
              />
            </Pressable>
            <Pressable
              onPress={handleReport}
              style={({ pressed }) => [
                styles.circleBtn,
                { backgroundColor: "rgba(0,0,0,0.4)", opacity: pressed ? 0.7 : 1 },
              ]}
            >
              <Feather name="flag" size={18} color="#fff" />
            </Pressable>
            <Pressable
              style={({ pressed }) => [
                styles.circleBtn,
                { backgroundColor: "rgba(0,0,0,0.4)", opacity: pressed ? 0.7 : 1 },
              ]}
            >
              <Feather name="share-2" size={18} color="#fff" />
            </Pressable>
          </View>
        </View>

        {listing.isAuction && timeLeft && (
          <View style={styles.auctionOverlay}>
            <Feather name="zap" size={12} color="#fff" />
            <Text style={styles.auctionOverlayText}>
              {formatAuctionType(listing.auctionType)} Auction · Ends in {timeLeft}
            </Text>
          </View>
        )}
      </View>

      <ScrollView
        style={styles.scroll}
        showsVerticalScrollIndicator={false}
        contentContainerStyle={styles.content}
      >
        <View style={styles.titleRow}>
          <Text style={[styles.title, { color: colors.text }]}>{listing.title}</Text>
          {listing.isAuction ? (
            <View style={styles.bidInfo}>
              <Text style={[styles.bidLabel, { color: colors.textTertiary }]}>
                {listing.auctionType === "dutch"
                  ? "Current price (dropping)"
                  : listing.auctionType === "reverse"
                  ? "Leading bid (rising)"
                  : "Current bid"}
              </Text>
              <Text style={[styles.price, { color: colors.tint }]}>
                {formatPrice(listing.currentBid ?? 0, currency)}
              </Text>
              <Text style={[styles.bidCount, { color: colors.textTertiary }]}>
                {listing.bidCount} bids
              </Text>
            </View>
          ) : (
            <Text style={[styles.price, { color: colors.tint }]}>
              {formatPrice(listing.price, currency)}
            </Text>
          )}
        </View>

        {listing.isAuction && listing.auctionType && listing.auctionType !== "standard" && (
          <View
            style={[
              styles.auctionTypeBanner,
              {
                backgroundColor:
                  listing.auctionType === "dutch" ? "#FFF7ED" : "#F0FDF4",
                borderColor:
                  listing.auctionType === "dutch" ? "#FB923C" : "#34D399",
              },
            ]}
          >
            <Feather
              name={listing.auctionType === "dutch" ? "trending-down" : "trending-up"}
              size={16}
              color={listing.auctionType === "dutch" ? "#92400E" : "#065F46"}
            />
            <View style={{ flex: 1 }}>
              <Text
                style={[
                  styles.auctionTypeTitle,
                  {
                    color:
                      listing.auctionType === "dutch" ? "#92400E" : "#065F46",
                  },
                ]}
              >
                {listing.auctionType === "dutch"
                  ? "Dutch Auction — Price Drops Over Time"
                  : "Reverse Auction — Lowest Bid Wins"}
              </Text>
              <Text
                style={[
                  styles.auctionTypeDesc,
                  {
                    color:
                      listing.auctionType === "dutch" ? "#B45309" : "#047857",
                  },
                ]}
              >
                {listing.auctionType === "dutch"
                  ? "The price decreases periodically. Buy now before it sells to another bidder."
                  : "Bid below competitors. The seller accepts the lowest qualifying bid when time expires."}
              </Text>
            </View>
          </View>
        )}

        {isEndingSoon && listing.isAuction && (
          <View
            style={[
              styles.antiSnipeBanner,
              { backgroundColor: "#FEF2F2", borderColor: "#FCA5A5" },
            ]}
          >
            <Feather name="shield" size={15} color="#991B1B" />
            <Text style={styles.antiSnipeText}>
              Anti-sniping active — a bid placed in the last 5 minutes extends the auction by 5 minutes.
            </Text>
          </View>
        )}

        <View style={styles.badges}>
          <View
            style={[
              styles.badge,
              { backgroundColor: getConditionColor(listing.condition) + "20" },
            ]}
          >
            <View
              style={[
                styles.dot,
                { backgroundColor: getConditionColor(listing.condition) },
              ]}
            />
            <Text
              style={[
                styles.badgeText,
                { color: getConditionColor(listing.condition) },
              ]}
            >
              {getConditionLabel(listing.condition)}
            </Text>
          </View>

          {listing.isFeatured && (
            <View style={[styles.badge, { backgroundColor: colors.tint + "20" }]}>
              <Feather name="star" size={11} color={colors.tint} />
              <Text style={[styles.badgeText, { color: colors.tint }]}>Featured</Text>
            </View>
          )}

          <View style={[styles.badge, { backgroundColor: colors.backgroundTertiary }]}>
            <Feather name="tag" size={11} color={colors.textTertiary} />
            <Text style={[styles.badgeText, { color: colors.textTertiary }]}>
              {currency}
            </Text>
          </View>
        </View>

        <View style={[styles.metaRow, { borderColor: colors.border }]}>
          <View style={styles.metaItem}>
            <Feather name="map-pin" size={14} color={colors.textTertiary} />
            <Text style={[styles.metaText, { color: colors.textSecondary }]}>
              {listing.location}
            </Text>
          </View>
          <View style={styles.metaItem}>
            <Feather name="clock" size={14} color={colors.textTertiary} />
            <Text style={[styles.metaText, { color: colors.textSecondary }]}>
              {formatRelativeTime(listing.createdAt)}
            </Text>
          </View>
          <View style={styles.metaItem}>
            <Feather name="eye" size={14} color={colors.textTertiary} />
            <Text style={[styles.metaText, { color: colors.textSecondary }]}>
              {listing.views.toLocaleString()} views
            </Text>
          </View>
        </View>

        <View style={styles.section}>
          <Text style={[styles.sectionTitle, { color: colors.text }]}>
            Description
          </Text>
          <Text style={[styles.description, { color: colors.textSecondary }]}>
            {listing.description}
          </Text>
        </View>

        {listing.tags.length > 0 && (
          <View style={styles.tags}>
            {listing.tags.map((tag) => (
              <View
                key={tag}
                style={[styles.tag, { backgroundColor: colors.backgroundTertiary }]}
              >
                <Text style={[styles.tagText, { color: colors.textSecondary }]}>
                  #{tag}
                </Text>
              </View>
            ))}
          </View>
        )}

        <Pressable
          onPress={() =>
            router.push({
              pathname: "/reviews/[userId]",
              params: { userId: listing.sellerId, name: listing.sellerName },
            })
          }
          style={[
            styles.sellerCard,
            { backgroundColor: colors.backgroundSecondary, borderColor: colors.border },
          ]}
        >
          <View style={[styles.sellerAvatar, { backgroundColor: colors.tint }]}>
            <Feather name="user" size={18} color="#fff" />
          </View>
          <View style={styles.sellerInfo}>
            <Text style={[styles.sellerName, { color: colors.text }]}>
              {listing.sellerName}
            </Text>
            <View style={styles.sellerRatingRow}>
              {[1, 2, 3, 4, 5].map((s) => (
                <Feather
                  key={s}
                  name="star"
                  size={11}
                  color={s <= 4 ? "#F59E0B" : colors.border}
                />
              ))}
              <Text style={[styles.sellerSub, { color: colors.textTertiary }]}>
                4.8 · See reviews
              </Text>
            </View>
          </View>
          <Feather name="chevron-right" size={18} color={colors.textTertiary} />
        </Pressable>

        <View style={{ height: isWeb ? 120 : 160 }} />
      </ScrollView>

      <View
        style={[
          styles.bottomBar,
          {
            paddingBottom: isWeb ? 34 : insets.bottom + 12,
            backgroundColor: colors.backgroundSecondary,
            borderTopColor: colors.border,
          },
        ]}
      >
        {listing.isAuction ? (
          <View style={styles.auctionActions}>
            <View style={styles.bidRow}>
              <View
                style={[
                  styles.bidInputContainer,
                  { backgroundColor: colors.backgroundTertiary, flex: 1 },
                ]}
              >
                <Text style={[styles.bidPrefix, { color: colors.textTertiary }]}>
                  {currency}
                </Text>
                <TextInput
                  style={[styles.bidInput, { color: colors.text }]}
                  placeholder={String(minNextBid)}
                  placeholderTextColor={colors.textTertiary}
                  value={bidAmount}
                  onChangeText={setBidAmount}
                  keyboardType="numeric"
                />
              </View>
              <Pressable
                onPress={handleBid}
                style={({ pressed }) => [
                  styles.bidBtn,
                  { backgroundColor: "#FFC220", opacity: pressed ? 0.85 : 1 },
                ]}
              >
                <Text style={{ fontSize: 18 }}>🔨</Text>
                <Text style={[styles.bidBtnText, { color: "#1A1A1A" }]}>Place Bid</Text>
              </Pressable>
            </View>

            <View style={styles.autoBidRow}>
              <View>
                <Text style={[styles.autoBidLabel, { color: colors.text }]}>
                  Auto-bid proxy
                </Text>
                <Text style={[styles.autoBidHint, { color: colors.textTertiary }]}>
                  Automatically outbid others up to your max
                </Text>
              </View>
              <Switch
                value={autoBid}
                onValueChange={(v) => {
                  Haptics.selectionAsync();
                  setAutoBid(v);
                }}
                trackColor={{ false: colors.border, true: colors.tint }}
                thumbColor="#fff"
              />
            </View>

            {autoBid && (
              <View
                style={[
                  styles.autoBidInput,
                  { backgroundColor: colors.backgroundTertiary },
                ]}
              >
                <Text style={[styles.bidPrefix, { color: colors.textTertiary }]}>
                  Max {currency}
                </Text>
                <TextInput
                  style={[styles.bidInput, { color: colors.text, flex: 1 }]}
                  placeholder="e.g. 5000"
                  placeholderTextColor={colors.textTertiary}
                  value={autoBidMax}
                  onChangeText={setAutoBidMax}
                  keyboardType="numeric"
                />
              </View>
            )}
          </View>
        ) : listing.sellerId !== user.id ? (
          <View style={styles.actionRow}>
            <Pressable
              onPress={handleContact}
              style={({ pressed }) => [
                styles.contactBtn,
                { backgroundColor: colors.tint, opacity: pressed ? 0.85 : 1 },
              ]}
            >
              <Feather name="message-circle" size={18} color="#fff" />
              <Text style={styles.contactBtnText}>Message Seller</Text>
            </Pressable>
            <Pressable
              style={({ pressed }) => [
                styles.offerBtn,
                { borderColor: colors.tint, opacity: pressed ? 0.8 : 1 },
              ]}
              onPress={() =>
                Alert.alert(
                  "Make Offer",
                  `Send the seller a price offer for ${formatPrice(listing.price, currency)}`,
                  [
                    { text: "Cancel", style: "cancel" },
                    {
                      text: "Send Offer",
                      onPress: () => {
                        const convId = startConversation(listing);
                        router.push({ pathname: "/conversation/[id]", params: { id: convId } });
                      },
                    },
                  ]
                )
              }
            >
              <Text style={[styles.offerBtnText, { color: colors.tint }]}>
                Make Offer
              </Text>
            </Pressable>
          </View>
        ) : (
          <View style={styles.ownerRow}>
            <Feather name="check-circle" size={18} color={colors.success} />
            <Text style={[styles.ownerText, { color: colors.textSecondary }]}>
              This is your listing
            </Text>
          </View>
        )}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  notFound: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: 12,
  },
  notFoundText: {
    fontSize: 18,
    fontFamily: "Inter_600SemiBold",
  },
  backLink: {
    fontSize: 15,
    fontFamily: "Inter_500Medium",
  },
  imageContainer: {
    height: 280,
    position: "relative",
  },
  imagePlaceholder: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
  },
  topBar: {
    position: "absolute",
    top: 0,
    left: 0,
    right: 0,
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    paddingHorizontal: 16,
  },
  topBarRight: {
    flexDirection: "row",
    gap: 8,
  },
  circleBtn: {
    width: 38,
    height: 38,
    borderRadius: 19,
    alignItems: "center",
    justifyContent: "center",
  },
  auctionOverlay: {
    position: "absolute",
    bottom: 12,
    left: 12,
    flexDirection: "row",
    alignItems: "center",
    gap: 5,
    backgroundColor: "rgba(10, 22, 40, 0.85)",
    paddingHorizontal: 10,
    paddingVertical: 6,
    borderRadius: 8,
  },
  auctionOverlayText: {
    fontSize: 12,
    fontFamily: "Inter_600SemiBold",
    color: "#fff",
  },
  scroll: { flex: 1 },
  content: {
    padding: 16,
    gap: 16,
  },
  titleRow: {
    gap: 8,
  },
  title: {
    fontSize: 22,
    fontFamily: "Inter_700Bold",
    lineHeight: 30,
  },
  price: {
    fontSize: 24,
    fontFamily: "Inter_700Bold",
  },
  bidInfo: {
    gap: 2,
  },
  bidLabel: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
  },
  bidCount: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
  },
  auctionTypeBanner: {
    flexDirection: "row",
    gap: 10,
    padding: 12,
    borderRadius: 10,
    borderWidth: 1,
    alignItems: "flex-start",
  },
  auctionTypeTitle: {
    fontSize: 13,
    fontFamily: "Inter_600SemiBold",
    marginBottom: 2,
  },
  auctionTypeDesc: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
    lineHeight: 18,
  },
  antiSnipeBanner: {
    flexDirection: "row",
    gap: 8,
    padding: 10,
    borderRadius: 10,
    borderWidth: 1,
    alignItems: "center",
  },
  antiSnipeText: {
    flex: 1,
    fontSize: 12,
    fontFamily: "Inter_400Regular",
    color: "#991B1B",
    lineHeight: 18,
  },
  badges: {
    flexDirection: "row",
    gap: 8,
    flexWrap: "wrap",
  },
  badge: {
    flexDirection: "row",
    alignItems: "center",
    gap: 5,
    paddingHorizontal: 10,
    paddingVertical: 5,
    borderRadius: 8,
  },
  dot: {
    width: 7,
    height: 7,
    borderRadius: 3.5,
  },
  badgeText: {
    fontSize: 12,
    fontFamily: "Inter_600SemiBold",
  },
  metaRow: {
    flexDirection: "row",
    justifyContent: "space-between",
    borderTopWidth: 1,
    borderBottomWidth: 1,
    paddingVertical: 12,
  },
  metaItem: {
    flexDirection: "row",
    alignItems: "center",
    gap: 5,
  },
  metaText: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
  },
  section: {
    gap: 8,
  },
  sectionTitle: {
    fontSize: 16,
    fontFamily: "Inter_700Bold",
  },
  description: {
    fontSize: 15,
    fontFamily: "Inter_400Regular",
    lineHeight: 24,
  },
  tags: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 8,
  },
  tag: {
    paddingHorizontal: 10,
    paddingVertical: 5,
    borderRadius: 6,
  },
  tagText: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
  },
  sellerCard: {
    flexDirection: "row",
    alignItems: "center",
    gap: 12,
    padding: 14,
    borderRadius: 14,
    borderWidth: 1,
  },
  sellerAvatar: {
    width: 42,
    height: 42,
    borderRadius: 21,
    alignItems: "center",
    justifyContent: "center",
  },
  sellerInfo: { flex: 1 },
  sellerName: {
    fontSize: 15,
    fontFamily: "Inter_600SemiBold",
  },
  sellerRatingRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 2,
    marginTop: 3,
  },
  sellerSub: {
    fontSize: 11,
    fontFamily: "Inter_400Regular",
    marginLeft: 4,
  },
  bottomBar: {
    borderTopWidth: 1,
    paddingHorizontal: 16,
    paddingTop: 12,
  },
  auctionActions: {
    gap: 10,
  },
  bidRow: {
    flexDirection: "row",
    gap: 10,
  },
  bidInputContainer: {
    flexDirection: "row",
    alignItems: "center",
    gap: 4,
    paddingHorizontal: 14,
    borderRadius: 12,
    height: 50,
  },
  bidPrefix: {
    fontSize: 13,
    fontFamily: "Inter_500Medium",
  },
  bidInput: {
    flex: 1,
    fontSize: 16,
    fontFamily: "Inter_700Bold",
  },
  bidBtn: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: 6,
    paddingHorizontal: 20,
    paddingVertical: 14,
    borderRadius: 12,
  },
  bidBtnText: {
    fontSize: 15,
    fontFamily: "Inter_600SemiBold",
    color: "#fff",
  },
  autoBidRow: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 12,
  },
  autoBidLabel: {
    fontSize: 14,
    fontFamily: "Inter_600SemiBold",
  },
  autoBidHint: {
    fontSize: 11,
    fontFamily: "Inter_400Regular",
    marginTop: 1,
  },
  autoBidInput: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
    paddingHorizontal: 14,
    borderRadius: 12,
    height: 46,
  },
  actionRow: {
    flexDirection: "row",
    gap: 10,
  },
  contactBtn: {
    flex: 1,
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: 8,
    paddingVertical: 14,
    borderRadius: 12,
  },
  contactBtnText: {
    fontSize: 15,
    fontFamily: "Inter_600SemiBold",
    color: "#fff",
  },
  offerBtn: {
    paddingHorizontal: 20,
    paddingVertical: 14,
    borderRadius: 12,
    borderWidth: 1.5,
  },
  offerBtnText: {
    fontSize: 15,
    fontFamily: "Inter_600SemiBold",
  },
  ownerRow: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: 8,
    paddingVertical: 14,
  },
  ownerText: {
    fontSize: 15,
    fontFamily: "Inter_500Medium",
  },
});
