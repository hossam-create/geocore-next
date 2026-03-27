import { Feather } from "@expo/vector-icons";
import { router, useLocalSearchParams } from "expo-router";
import React from "react";
import {
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

interface Review {
  id: string;
  reviewerName: string;
  rating: number;
  comment: string;
  date: string;
  listingTitle: string;
}

const MOCK_REVIEWS: Review[] = [
  {
    id: "r-1",
    reviewerName: "Sarah M.",
    rating: 5,
    comment: "Excellent seller! Item was exactly as described and shipped very quickly. Would buy again.",
    date: new Date(Date.now() - 86400000 * 5).toISOString(),
    listingTitle: "iPhone 14 Pro Max",
  },
  {
    id: "r-2",
    reviewerName: "Khalid A.",
    rating: 4,
    comment: "Good experience overall. Item arrived in great condition. Minor delay in responding but resolved quickly.",
    date: new Date(Date.now() - 86400000 * 12).toISOString(),
    listingTitle: "PlayStation 5",
  },
  {
    id: "r-3",
    reviewerName: "Layla H.",
    rating: 5,
    comment: "Outstanding! Went above and beyond. Included original box, cables, and even a screen protector.",
    date: new Date(Date.now() - 86400000 * 20).toISOString(),
    listingTitle: "AirPods Pro 2nd Gen",
  },
  {
    id: "r-4",
    reviewerName: "Omar Y.",
    rating: 5,
    comment: "Very professional seller. Honest description, no surprises. Fast communication. Highly recommended.",
    date: new Date(Date.now() - 86400000 * 34).toISOString(),
    listingTitle: "MacBook Pro M2",
  },
  {
    id: "r-5",
    reviewerName: "Nadia R.",
    rating: 4,
    comment: "Great deal. Smooth transaction. Small cosmetic scratch not mentioned but seller gave a small discount.",
    date: new Date(Date.now() - 86400000 * 60).toISOString(),
    listingTitle: "Samsung Galaxy S23 Ultra",
  },
];

const AVG_RATING = 4.8;
const BREAKDOWN = [5, 4, 3, 2, 1].map((star) => ({
  star,
  count: MOCK_REVIEWS.filter((r) => r.rating === star).length,
}));

function StarRow({ rating, size = 14 }: { rating: number; size?: number }) {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  return (
    <View style={{ flexDirection: "row", gap: 2 }}>
      {[1, 2, 3, 4, 5].map((s) => (
        <Feather
          key={s}
          name="star"
          size={size}
          color={s <= rating ? "#F59E0B" : colors.border}
        />
      ))}
    </View>
  );
}

export default function ReviewsScreen() {
  const { userId, name } = useLocalSearchParams<{
    userId: string;
    name: string;
  }>();
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const insets = useSafeAreaInsets();
  const isWeb = Platform.OS === "web";
  const topPad = isWeb ? 67 : insets.top;

  const formatDate = (dateStr: string) =>
    new Date(dateStr).toLocaleDateString("en-AE", {
      day: "numeric",
      month: "short",
      year: "numeric",
    });

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
          style={({ pressed }) => [
            styles.backBtn,
            pressed && { opacity: 0.7 },
          ]}
        >
          <Feather name="arrow-left" size={22} color={colors.text} />
        </Pressable>
        <View>
          <Text style={[styles.headerTitle, { color: colors.text }]}>
            Reviews
          </Text>
          <Text style={[styles.headerSub, { color: colors.textTertiary }]}>
            {name ?? "Seller"}
          </Text>
        </View>
      </View>

      <ScrollView
        showsVerticalScrollIndicator={false}
        contentContainerStyle={styles.content}
      >
        <View
          style={[
            styles.summaryCard,
            {
              backgroundColor: colors.backgroundSecondary,
              borderColor: colors.border,
            },
          ]}
        >
          <View style={styles.summaryLeft}>
            <Text style={[styles.avgRating, { color: colors.text }]}>
              {AVG_RATING}
            </Text>
            <StarRow rating={Math.round(AVG_RATING)} size={18} />
            <Text style={[styles.reviewCount, { color: colors.textTertiary }]}>
              {MOCK_REVIEWS.length} reviews
            </Text>
          </View>

          <View style={styles.summaryRight}>
            {BREAKDOWN.map(({ star, count }) => (
              <View key={star} style={styles.breakdownRow}>
                <Text
                  style={[styles.breakdownStar, { color: colors.textTertiary }]}
                >
                  {star}
                </Text>
                <Feather name="star" size={11} color="#F59E0B" />
                <View
                  style={[
                    styles.breakdownBar,
                    { backgroundColor: colors.backgroundTertiary },
                  ]}
                >
                  <View
                    style={[
                      styles.breakdownFill,
                      {
                        backgroundColor: colors.tint,
                        width: `${(count / MOCK_REVIEWS.length) * 100}%`,
                      },
                    ]}
                  />
                </View>
                <Text
                  style={[styles.breakdownCount, { color: colors.textTertiary }]}
                >
                  {count}
                </Text>
              </View>
            ))}
          </View>
        </View>

        {MOCK_REVIEWS.map((review) => (
          <View
            key={review.id}
            style={[
              styles.reviewCard,
              {
                backgroundColor: colors.backgroundSecondary,
                borderColor: colors.border,
              },
            ]}
          >
            <View style={styles.reviewHeader}>
              <View style={[styles.reviewAvatar, { backgroundColor: colors.tint }]}>
                <Text style={styles.reviewAvatarText}>
                  {review.reviewerName[0]}
                </Text>
              </View>
              <View style={{ flex: 1 }}>
                <Text style={[styles.reviewerName, { color: colors.text }]}>
                  {review.reviewerName}
                </Text>
                <View style={styles.reviewMeta}>
                  <StarRow rating={review.rating} size={12} />
                  <Text
                    style={[styles.reviewDate, { color: colors.textTertiary }]}
                  >
                    {formatDate(review.date)}
                  </Text>
                </View>
              </View>
            </View>

            <Text style={[styles.reviewComment, { color: colors.textSecondary }]}>
              {review.comment}
            </Text>

            <View
              style={[
                styles.reviewListing,
                { backgroundColor: colors.backgroundTertiary },
              ]}
            >
              <Feather name="tag" size={11} color={colors.textTertiary} />
              <Text
                style={[styles.reviewListingText, { color: colors.textTertiary }]}
              >
                {review.listingTitle}
              </Text>
            </View>
          </View>
        ))}

        <View style={{ height: isWeb ? 34 : 100 }} />
      </ScrollView>
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
    gap: 12,
  },
  backBtn: {
    width: 36,
    height: 36,
    alignItems: "center",
    justifyContent: "center",
  },
  headerTitle: {
    fontSize: 20,
    fontFamily: "Inter_700Bold",
  },
  headerSub: {
    fontSize: 13,
    fontFamily: "Inter_400Regular",
  },
  content: {
    padding: 16,
    gap: 16,
  },
  summaryCard: {
    borderRadius: 16,
    borderWidth: 1,
    padding: 20,
    flexDirection: "row",
    gap: 20,
    alignItems: "center",
  },
  summaryLeft: {
    alignItems: "center",
    gap: 6,
  },
  avgRating: {
    fontSize: 48,
    fontFamily: "Inter_700Bold",
    lineHeight: 54,
  },
  reviewCount: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
    marginTop: 2,
  },
  summaryRight: {
    flex: 1,
    gap: 6,
  },
  breakdownRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
  },
  breakdownStar: {
    fontSize: 12,
    fontFamily: "Inter_500Medium",
    width: 10,
    textAlign: "right",
  },
  breakdownBar: {
    flex: 1,
    height: 6,
    borderRadius: 3,
    overflow: "hidden",
  },
  breakdownFill: {
    height: "100%",
    borderRadius: 3,
  },
  breakdownCount: {
    fontSize: 11,
    fontFamily: "Inter_400Regular",
    width: 14,
    textAlign: "right",
  },
  reviewCard: {
    borderRadius: 14,
    borderWidth: 1,
    padding: 16,
    gap: 12,
  },
  reviewHeader: {
    flexDirection: "row",
    gap: 12,
    alignItems: "flex-start",
  },
  reviewAvatar: {
    width: 38,
    height: 38,
    borderRadius: 19,
    alignItems: "center",
    justifyContent: "center",
  },
  reviewAvatarText: {
    fontSize: 16,
    fontFamily: "Inter_700Bold",
    color: "#fff",
  },
  reviewerName: {
    fontSize: 15,
    fontFamily: "Inter_600SemiBold",
  },
  reviewMeta: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
    marginTop: 3,
  },
  reviewDate: {
    fontSize: 11,
    fontFamily: "Inter_400Regular",
  },
  reviewComment: {
    fontSize: 14,
    fontFamily: "Inter_400Regular",
    lineHeight: 22,
  },
  reviewListing: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
    paddingHorizontal: 10,
    paddingVertical: 6,
    borderRadius: 8,
    alignSelf: "flex-start",
  },
  reviewListingText: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
  },
});
