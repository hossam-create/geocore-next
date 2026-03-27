import { Feather } from "@expo/vector-icons";
import { router, useLocalSearchParams } from "expo-router";
import React, { useEffect, useState } from "react";
import {
  ActivityIndicator,
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

import ListingCard from "@/components/ListingCard";
import Colors from "@/constants/colors";
import { storesAPI } from "@/utils/api";

export default function StorefrontDetailScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const insets = useSafeAreaInsets();
  const isWeb = Platform.OS === "web";
  const topPad = isWeb ? 67 : insets.top;
  const { slug } = useLocalSearchParams<{ slug: string }>();

  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  useEffect(() => {
    if (!slug) return;
    storesAPI
      .getBySlug(slug)
      .then((res) => setData(res.data?.data ?? res.data))
      .catch(() => setError(true))
      .finally(() => setLoading(false));
  }, [slug]);

  if (loading) {
    return (
      <View style={[styles.container, { backgroundColor: colors.background }]}>
        <View style={[styles.header, { paddingTop: topPad + 10 }]}>
          <Pressable onPress={() => router.back()} style={styles.backBtn}>
            <Feather name="arrow-left" size={22} color={colors.text} />
          </Pressable>
        </View>
        <View style={styles.centered}>
          <ActivityIndicator size="large" color="#0071CE" />
        </View>
      </View>
    );
  }

  if (error || !data) {
    return (
      <View style={[styles.container, { backgroundColor: colors.background }]}>
        <View style={[styles.header, { paddingTop: topPad + 10 }]}>
          <Pressable onPress={() => router.back()} style={styles.backBtn}>
            <Feather name="arrow-left" size={22} color={colors.text} />
          </Pressable>
        </View>
        <View style={styles.centered}>
          <Feather name="alert-circle" size={40} color={colors.textSecondary} />
          <Text style={[styles.emptyText, { color: colors.textSecondary }]}>
            Storefront not found
          </Text>
        </View>
      </View>
    );
  }

  const storefront = data?.storefront ?? data;
  const listings: any[] = data?.listings ?? [];
  const initial = storefront.name?.[0]?.toUpperCase() ?? "S";

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <View style={[styles.header, { paddingTop: topPad + 10 }]}>
        <Pressable onPress={() => router.back()} style={styles.backBtn}>
          <Feather name="arrow-left" size={22} color={colors.text} />
        </Pressable>
        <Text style={[styles.headerTitle, { color: colors.text }]} numberOfLines={1}>
          {storefront.name}
        </Text>
        <View style={{ width: 38 }} />
      </View>

      <ScrollView showsVerticalScrollIndicator={false} contentContainerStyle={styles.scroll}>
        <View style={[styles.banner, { backgroundColor: "#0071CE" }]} />

        <View style={[styles.storeInfo, { backgroundColor: colors.backgroundSecondary, borderColor: colors.border }]}>
          <View style={[styles.avatar, { backgroundColor: "#FFC220" }]}>
            <Text style={styles.avatarText}>{initial}</Text>
          </View>
          <Text style={[styles.storeName, { color: colors.text }]}>{storefront.name}</Text>
          {storefront.description ? (
            <Text style={[styles.storeDesc, { color: colors.textSecondary }]}>
              {storefront.description}
            </Text>
          ) : null}
          <View style={styles.statsRow}>
            {(storefront.views ?? 0) > 0 && (
              <View style={styles.stat}>
                <Feather name="eye" size={13} color={colors.textSecondary} />
                <Text style={[styles.statText, { color: colors.textSecondary }]}>
                  {(storefront.views ?? 0).toLocaleString()} views
                </Text>
              </View>
            )}
            {listings.length > 0 && (
              <View style={styles.stat}>
                <Feather name="package" size={13} color={colors.textSecondary} />
                <Text style={[styles.statText, { color: colors.textSecondary }]}>
                  {listings.length} listings
                </Text>
              </View>
            )}
          </View>
        </View>

        {listings.length > 0 ? (
          <View style={styles.listingsSection}>
            <Text style={[styles.sectionTitle, { color: colors.text }]}>Listings</Text>
            <View style={styles.grid}>
              {listings.map((item: any) => (
                <View key={item.id} style={styles.cardWrap}>
                  <ListingCard listing={item} />
                </View>
              ))}
            </View>
          </View>
        ) : (
          <View style={styles.centered}>
            <Feather name="inbox" size={36} color={colors.textSecondary} />
            <Text style={[styles.emptyText, { color: colors.textSecondary }]}>
              No listings yet
            </Text>
          </View>
        )}
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: 16,
    paddingBottom: 12,
  },
  backBtn: { padding: 8 },
  headerTitle: { fontSize: 16, fontFamily: "Inter_700Bold", flex: 1, textAlign: "center" },
  scroll: { paddingBottom: 40 },
  banner: { height: 140 },
  storeInfo: {
    marginHorizontal: 16,
    marginTop: -24,
    borderRadius: 16,
    borderWidth: 1,
    padding: 16,
    gap: 6,
  },
  avatar: {
    width: 56,
    height: 56,
    borderRadius: 14,
    borderWidth: 3,
    borderColor: "#fff",
    alignItems: "center",
    justifyContent: "center",
    marginTop: -36,
    marginBottom: 4,
  },
  avatarText: { fontSize: 22, fontFamily: "Inter_700Bold", color: "#1A1A1A" },
  storeName: { fontSize: 18, fontFamily: "Inter_700Bold" },
  storeDesc: { fontSize: 13, lineHeight: 20 },
  statsRow: { flexDirection: "row", gap: 16, marginTop: 6 },
  stat: { flexDirection: "row", alignItems: "center", gap: 4 },
  statText: { fontSize: 12 },
  listingsSection: { padding: 16, gap: 12 },
  sectionTitle: { fontSize: 16, fontFamily: "Inter_700Bold" },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: 12 },
  cardWrap: { width: "47%" },
  centered: { padding: 40, alignItems: "center", gap: 10 },
  emptyText: { fontSize: 15, fontFamily: "Inter_600SemiBold", textAlign: "center" },
});
