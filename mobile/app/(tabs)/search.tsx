import { Feather } from "@expo/vector-icons";
import * as Haptics from "expo-haptics";
import React, { useMemo, useState } from "react";
import {
  FlatList,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
  useColorScheme,
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import ListingCard from "@/components/ListingCard";
import Colors from "@/constants/colors";
import { useAppContext } from "@/context/AppContext";

const TYPE_FILTERS: { id: string; label: string }[] = [
  { id: "all", label: "All" },
  { id: "auction", label: "Auction" },
  { id: "buy-now", label: "Buy Now" },
  { id: "featured", label: "Featured" },
];

const SORT_OPTIONS: { id: string; label: string; icon: string }[] = [
  { id: "newest", label: "Newest", icon: "clock" },
  { id: "price_asc", label: "Price ↑", icon: "trending-up" },
  { id: "price_desc", label: "Price ↓", icon: "trending-down" },
  { id: "popular", label: "Popular", icon: "eye" },
];

export default function SearchScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const insets = useSafeAreaInsets();
  const isWeb = Platform.OS === "web";
  const topPad = isWeb ? 67 : insets.top;
  const { listings } = useAppContext();

  const [query, setQuery] = useState("");
  const [activeFilter, setActiveFilter] = useState("all");
  const [activeSort, setActiveSort] = useState("newest");
  const [showSort, setShowSort] = useState(false);

  const results = useMemo(() => {
    let filtered = [...listings];

    if (query.trim()) {
      const q = query.toLowerCase();
      filtered = filtered.filter(
        (l) =>
          l.title.toLowerCase().includes(q) ||
          l.description.toLowerCase().includes(q) ||
          l.location.toLowerCase().includes(q) ||
          l.tags.some((t) => t.toLowerCase().includes(q))
      );
    }

    if (activeFilter === "auction") {
      filtered = filtered.filter((l) => l.isAuction);
    } else if (activeFilter === "buy-now") {
      filtered = filtered.filter((l) => !l.isAuction);
    } else if (activeFilter === "featured") {
      filtered = filtered.filter((l) => l.isFeatured);
    }

    switch (activeSort) {
      case "newest":
        filtered.sort(
          (a, b) =>
            new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
        );
        break;
      case "price_asc":
        filtered.sort((a, b) => {
          const pa = a.isAuction ? (a.currentBid ?? 0) : a.price;
          const pb = b.isAuction ? (b.currentBid ?? 0) : b.price;
          return pa - pb;
        });
        break;
      case "price_desc":
        filtered.sort((a, b) => {
          const pa = a.isAuction ? (a.currentBid ?? 0) : a.price;
          const pb = b.isAuction ? (b.currentBid ?? 0) : b.price;
          return pb - pa;
        });
        break;
      case "popular":
        filtered.sort((a, b) => b.views - a.views);
        break;
    }

    return filtered;
  }, [listings, query, activeFilter, activeSort]);

  const activeSortLabel =
    SORT_OPTIONS.find((s) => s.id === activeSort)?.label ?? "Newest";

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <View
        style={[
          styles.header,
          {
            paddingTop: topPad + 8,
            backgroundColor: colors.backgroundSecondary,
            borderBottomColor: colors.border,
          },
        ]}
      >
        <View style={styles.searchRow}>
          <View
            style={[
              styles.searchBar,
              { backgroundColor: colors.backgroundTertiary, flex: 1 },
            ]}
          >
            <Feather name="search" size={16} color={colors.textTertiary} />
            <TextInput
              style={[styles.input, { color: colors.text }]}
              placeholder="Search listings, locations..."
              placeholderTextColor={colors.textTertiary}
              value={query}
              onChangeText={setQuery}
              autoCapitalize="none"
              returnKeyType="search"
            />
            {query.length > 0 && (
              <Pressable onPress={() => setQuery("")}>
                <Feather name="x" size={16} color={colors.textTertiary} />
              </Pressable>
            )}
          </View>

          <Pressable
            onPress={() => {
              Haptics.selectionAsync();
              setShowSort((v) => !v);
            }}
            style={[
              styles.sortBtn,
              {
                backgroundColor: showSort
                  ? colors.tint
                  : colors.backgroundTertiary,
              },
            ]}
          >
            <Feather
              name="sliders"
              size={16}
              color={showSort ? "#fff" : colors.textSecondary}
            />
          </Pressable>
        </View>

        <ScrollView
          horizontal
          showsHorizontalScrollIndicator={false}
          contentContainerStyle={styles.filters}
        >
          {TYPE_FILTERS.map((f) => (
            <Pressable
              key={f.id}
              onPress={() => {
                Haptics.selectionAsync();
                setActiveFilter(f.id);
              }}
              style={[
                styles.filterChip,
                {
                  backgroundColor:
                    activeFilter === f.id
                      ? colors.tint
                      : colors.backgroundTertiary,
                },
              ]}
            >
              <Text
                style={[
                  styles.filterLabel,
                  {
                    color:
                      activeFilter === f.id ? "#fff" : colors.textSecondary,
                  },
                ]}
              >
                {f.label}
              </Text>
            </Pressable>
          ))}
        </ScrollView>

        {showSort && (
          <View style={styles.sortRow}>
            {SORT_OPTIONS.map((s) => (
              <Pressable
                key={s.id}
                onPress={() => {
                  Haptics.selectionAsync();
                  setActiveSort(s.id);
                  setShowSort(false);
                }}
                style={[
                  styles.sortChip,
                  {
                    backgroundColor:
                      activeSort === s.id
                        ? colors.tint + "22"
                        : colors.backgroundTertiary,
                    borderColor:
                      activeSort === s.id ? colors.tint : colors.border,
                  },
                ]}
              >
                <Feather
                  name={s.icon as any}
                  size={13}
                  color={activeSort === s.id ? colors.tint : colors.textSecondary}
                />
                <Text
                  style={[
                    styles.sortLabel,
                    {
                      color:
                        activeSort === s.id ? colors.tint : colors.textSecondary,
                    },
                  ]}
                >
                  {s.label}
                </Text>
              </Pressable>
            ))}
          </View>
        )}
      </View>

      <FlatList
        data={results}
        keyExtractor={(item) => item.id}
        contentContainerStyle={styles.list}
        contentInsetAdjustmentBehavior="automatic"
        showsVerticalScrollIndicator={false}
        renderItem={({ item }) => <ListingCard listing={item} />}
        ListHeaderComponent={
          <View style={styles.resultsMeta}>
            <Text style={[styles.resultsCount, { color: colors.textTertiary }]}>
              {results.length} result{results.length !== 1 ? "s" : ""}
            </Text>
            <Text style={[styles.resultsSortLabel, { color: colors.tint }]}>
              {activeSortLabel}
            </Text>
          </View>
        }
        ListEmptyComponent={
          <View style={styles.empty}>
            <Feather name="search" size={40} color={colors.textTertiary} />
            <Text style={[styles.emptyTitle, { color: colors.text }]}>
              No results found
            </Text>
            <Text style={[styles.emptyText, { color: colors.textTertiary }]}>
              Try different keywords or adjust your filters
            </Text>
          </View>
        }
        ListFooterComponent={<View style={{ height: isWeb ? 34 : 100 }} />}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  header: {
    borderBottomWidth: 1,
    paddingHorizontal: 16,
    paddingBottom: 12,
    gap: 10,
  },
  searchRow: {
    flexDirection: "row",
    gap: 8,
    alignItems: "center",
  },
  searchBar: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    paddingHorizontal: 14,
    paddingVertical: 12,
    borderRadius: 14,
  },
  input: {
    flex: 1,
    fontSize: 15,
    fontFamily: "Inter_400Regular",
  },
  sortBtn: {
    width: 44,
    height: 44,
    borderRadius: 14,
    alignItems: "center",
    justifyContent: "center",
  },
  filters: {
    gap: 8,
    paddingRight: 4,
  },
  filterChip: {
    paddingHorizontal: 14,
    paddingVertical: 7,
    borderRadius: 20,
  },
  filterLabel: {
    fontSize: 13,
    fontFamily: "Inter_500Medium",
  },
  sortRow: {
    flexDirection: "row",
    gap: 8,
    flexWrap: "wrap",
  },
  sortChip: {
    flexDirection: "row",
    alignItems: "center",
    gap: 5,
    paddingHorizontal: 12,
    paddingVertical: 7,
    borderRadius: 20,
    borderWidth: 1,
  },
  sortLabel: {
    fontSize: 13,
    fontFamily: "Inter_500Medium",
  },
  list: {
    paddingHorizontal: 16,
    paddingTop: 12,
  },
  resultsMeta: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    marginBottom: 8,
  },
  resultsCount: {
    fontSize: 13,
    fontFamily: "Inter_400Regular",
  },
  resultsSortLabel: {
    fontSize: 12,
    fontFamily: "Inter_600SemiBold",
  },
  empty: {
    alignItems: "center",
    justifyContent: "center",
    paddingTop: 80,
    gap: 12,
  },
  emptyTitle: {
    fontSize: 18,
    fontFamily: "Inter_600SemiBold",
  },
  emptyText: {
    fontSize: 14,
    fontFamily: "Inter_400Regular",
    textAlign: "center",
    paddingHorizontal: 40,
  },
});
