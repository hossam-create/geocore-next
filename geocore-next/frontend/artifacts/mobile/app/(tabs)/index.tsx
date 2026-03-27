import { Feather, Ionicons } from "@expo/vector-icons";
import * as Haptics from "expo-haptics";
import { router } from "expo-router";
import React, { useState } from "react";
import {
  FlatList,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
  useColorScheme,
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import AuctionCard from "@/components/AuctionCard";
import ListingCard from "@/components/ListingCard";
import Colors from "@/constants/colors";
import { useAppContext, type ListingCategory } from "@/context/AppContext";

const CATEGORIES: { id: ListingCategory | "all"; label: string; emoji: string }[] = [
  { id: "vehicles", label: "Vehicles", emoji: "🚗" },
  { id: "real-estate", label: "Real Estate", emoji: "🏠" },
  { id: "electronics", label: "Electronics", emoji: "📱" },
  { id: "fashion", label: "Clothing", emoji: "👕" },
  { id: "furniture", label: "Furniture", emoji: "🛋️" },
  { id: "sports", label: "Sports", emoji: "⚽" },
  { id: "services", label: "Services", emoji: "🔧" },
  { id: "all", label: "More", emoji: "🔮" },
];

export default function HomeScreen() {
  const colorScheme = useColorScheme();
  const isDark = colorScheme === "dark";
  const colors = Colors[colorScheme ?? "light"];
  const insets = useSafeAreaInsets();
  const { listings, user } = useAppContext();
  const [activeCategory, setActiveCategory] = useState<ListingCategory | "all">("all");
  const isWeb = Platform.OS === "web";
  const topPad = isWeb ? 67 : insets.top;

  const auctions = listings.filter((l) => l.isAuction);
  const filtered =
    activeCategory === "all"
      ? listings
      : listings.filter((l) => l.category === activeCategory);

  const gridPairs: [typeof listings[0], typeof listings[0] | null][] = [];
  for (let i = 0; i < filtered.length; i += 2) {
    gridPairs.push([filtered[i], filtered[i + 1] ?? null]);
  }

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <View style={[styles.header, { paddingTop: topPad + 10 }]}>
        <View style={styles.headerTop}>
          <Text style={styles.logo}>GeoCore</Text>
          <View style={styles.headerIcons}>
            <Pressable
              onPress={() => router.push("/notifications")}
              style={({ pressed }) => [styles.headerIcon, pressed && { opacity: 0.7 }]}
            >
              <Ionicons name="notifications-outline" size={24} color="#fff" />
            </Pressable>
            <Pressable
              onPress={() => router.push("/(tabs)/profile" as any)}
              style={({ pressed }) => [styles.headerIcon, pressed && { opacity: 0.7 }]}
            >
              <Ionicons name="person-outline" size={24} color="#fff" />
            </Pressable>
          </View>
        </View>

        <Pressable
          onPress={() => router.push("/(tabs)/search" as any)}
          style={({ pressed }) => [styles.searchBar, pressed && { opacity: 0.93 }]}
        >
          <Feather name="search" size={18} color="#888" />
          <Text style={styles.searchPlaceholder}>Search for anything...</Text>
        </Pressable>
      </View>

      <ScrollView
        style={{ flex: 1 }}
        showsVerticalScrollIndicator={false}
        contentContainerStyle={styles.scrollContent}
      >
        <View style={styles.section}>
          <Text style={[styles.sectionTitle, { color: colors.text }]}>Categories</Text>
          <ScrollView
            horizontal
            showsHorizontalScrollIndicator={false}
            contentContainerStyle={styles.categoriesRow}
          >
            {CATEGORIES.map((cat) => (
              <Pressable
                key={cat.id}
                onPress={() => {
                  Haptics.selectionAsync();
                  setActiveCategory(cat.id as ListingCategory | "all");
                }}
                style={styles.categoryItem}
              >
                <View
                  style={[
                    styles.categoryCircle,
                    {
                      backgroundColor:
                        activeCategory === cat.id
                          ? colors.tint
                          : isDark
                          ? colors.backgroundTertiary
                          : "#E3F2FD",
                    },
                  ]}
                >
                  <Text style={styles.categoryEmoji}>{cat.emoji}</Text>
                </View>
                <Text
                  style={[
                    styles.categoryLabel,
                    {
                      color:
                        activeCategory === cat.id ? colors.tint : colors.textSecondary,
                      fontFamily:
                        activeCategory === cat.id
                          ? "Inter_600SemiBold"
                          : "Inter_400Regular",
                    },
                  ]}
                >
                  {cat.label}
                </Text>
              </Pressable>
            ))}
          </ScrollView>
        </View>

        {activeCategory === "all" && auctions.length > 0 && (
          <View style={styles.section}>
            <View style={styles.sectionHeader}>
              <View style={styles.sectionTitleRow}>
                <Text style={styles.auctionDot}>⚡</Text>
                <Text style={[styles.sectionTitle, { color: colors.text }]}>
                  Live Auctions
                </Text>
              </View>
              <Pressable onPress={() => router.push("/(tabs)/search" as any)}>
                <Text style={[styles.seeAll, { color: colors.tint }]}>See all →</Text>
              </Pressable>
            </View>
            <ScrollView
              horizontal
              showsHorizontalScrollIndicator={false}
              contentContainerStyle={styles.horizontalRow}
            >
              {auctions.map((l) => (
                <AuctionCard key={l.id} listing={l} />
              ))}
            </ScrollView>
          </View>
        )}

        <View style={styles.section}>
          <View style={styles.sectionHeader}>
            <Text style={[styles.sectionTitle, { color: colors.text }]}>
              {activeCategory === "all" ? "All Listings" : CATEGORIES.find(c => c.id === activeCategory)?.label ?? "Listings"}
            </Text>
            <Text style={[styles.listingCount, { color: colors.textTertiary }]}>
              {filtered.length} items
            </Text>
          </View>

          {gridPairs.map(([left, right], index) => (
            <View key={index} style={styles.gridRow}>
              <View style={styles.gridCell}>
                <ListingCard listing={left} />
              </View>
              <View style={styles.gridCell}>
                {right ? (
                  <ListingCard listing={right} />
                ) : (
                  <View style={styles.gridCell} />
                )}
              </View>
            </View>
          ))}

          {filtered.length === 0 && (
            <View style={styles.empty}>
              <Text style={{ fontSize: 40 }}>🔍</Text>
              <Text style={[styles.emptyText, { color: colors.textTertiary }]}>
                No listings in this category yet
              </Text>
            </View>
          )}
        </View>

        <View style={{ height: isWeb ? 34 : 100 }} />
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  header: {
    backgroundColor: "#0071CE",
    paddingHorizontal: 16,
    paddingBottom: 14,
    gap: 12,
  },
  headerTop: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
  },
  logo: {
    color: "#FFC220",
    fontSize: 24,
    fontFamily: "Inter_700Bold",
  },
  headerIcons: {
    flexDirection: "row",
    gap: 16,
  },
  headerIcon: {
    padding: 2,
  },
  searchBar: {
    backgroundColor: "#fff",
    borderRadius: 8,
    paddingHorizontal: 14,
    paddingVertical: 12,
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
  },
  searchPlaceholder: {
    color: "#888",
    fontSize: 15,
    fontFamily: "Inter_400Regular",
    flex: 1,
  },
  scrollContent: {
    paddingTop: 8,
  },
  section: {
    paddingHorizontal: 14,
    marginBottom: 20,
  },
  sectionHeader: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    marginBottom: 12,
  },
  sectionTitleRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 5,
  },
  auctionDot: {
    fontSize: 16,
  },
  sectionTitle: {
    fontSize: 17,
    fontFamily: "Inter_700Bold",
    marginBottom: 12,
  },
  seeAll: {
    fontSize: 13,
    fontFamily: "Inter_500Medium",
  },
  listingCount: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
    marginBottom: 12,
  },
  categoriesRow: {
    gap: 16,
    paddingRight: 8,
  },
  categoryItem: {
    alignItems: "center",
    width: 60,
    gap: 5,
  },
  categoryCircle: {
    width: 54,
    height: 54,
    borderRadius: 27,
    alignItems: "center",
    justifyContent: "center",
  },
  categoryEmoji: {
    fontSize: 24,
  },
  categoryLabel: {
    fontSize: 11,
    textAlign: "center",
  },
  horizontalRow: {
    paddingRight: 8,
  },
  gridRow: {
    flexDirection: "row",
    gap: 10,
    marginBottom: 2,
  },
  gridCell: {
    flex: 1,
  },
  empty: {
    alignItems: "center",
    paddingTop: 40,
    gap: 12,
  },
  emptyText: {
    fontSize: 14,
    fontFamily: "Inter_400Regular",
    textAlign: "center",
  },
});
