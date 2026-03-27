import { Feather } from "@expo/vector-icons";
import { router } from "expo-router";
import React, { useEffect, useState } from "react";
import {
  ActivityIndicator,
  FlatList,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  View,
  useColorScheme,
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import Colors from "@/constants/colors";
import { storesAPI } from "@/utils/api";

interface Store {
  id: string;
  slug: string;
  name: string;
  description: string;
  logo_url?: string;
  banner_url?: string;
  views?: number;
  created_at: string;
}

function StoreCard({ store, colors }: { store: Store; colors: any }) {
  const initial = store.name?.[0]?.toUpperCase() ?? "S";
  return (
    <Pressable
      style={({ pressed }) => [
        styles.card,
        {
          backgroundColor: colors.backgroundSecondary,
          borderColor: colors.border,
          opacity: pressed ? 0.85 : 1,
        },
      ]}
      onPress={() => router.push(`/storefront/${store.slug}` as any)}
    >
      <View style={[styles.cardBanner, { backgroundColor: "#0071CE" }]} />
      <View style={styles.cardBody}>
        <View style={[styles.avatar, { backgroundColor: "#FFC220" }]}>
          <Text style={styles.avatarText}>{initial}</Text>
        </View>
        <Text style={[styles.storeName, { color: colors.text }]} numberOfLines={1}>
          {store.name}
        </Text>
        <Text style={[styles.storeDesc, { color: colors.textSecondary }]} numberOfLines={2}>
          {store.description || "GCC Marketplace seller"}
        </Text>
        {(store.views ?? 0) > 0 && (
          <View style={styles.viewsRow}>
            <Feather name="eye" size={11} color={colors.textSecondary} />
            <Text style={[styles.viewsText, { color: colors.textSecondary }]}>
              {(store.views ?? 0).toLocaleString()} views
            </Text>
          </View>
        )}
      </View>
    </Pressable>
  );
}

export default function StoresScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const insets = useSafeAreaInsets();
  const isWeb = Platform.OS === "web";
  const topPad = isWeb ? 67 : insets.top;

  const [stores, setStores] = useState<Store[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  useEffect(() => {
    storesAPI
      .list()
      .then((res) => {
        const data = res.data?.data ?? res.data ?? [];
        setStores(Array.isArray(data) ? data : []);
      })
      .catch(() => setError(true))
      .finally(() => setLoading(false));
  }, []);

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <View style={[styles.header, { paddingTop: topPad + 10 }]}>
        <Pressable onPress={() => router.back()} style={styles.backBtn}>
          <Feather name="arrow-left" size={22} color={colors.text} />
        </Pressable>
        <Text style={[styles.title, { color: colors.text }]}>Seller Storefronts</Text>
        <View style={{ width: 38 }} />
      </View>

      {loading && (
        <View style={styles.centered}>
          <ActivityIndicator size="large" color="#0071CE" />
        </View>
      )}

      {!loading && error && (
        <View style={styles.centered}>
          <Feather name="alert-circle" size={40} color={colors.textSecondary} />
          <Text style={[styles.emptyText, { color: colors.textSecondary }]}>
            Could not load storefronts
          </Text>
        </View>
      )}

      {!loading && !error && stores.length === 0 && (
        <View style={styles.centered}>
          <Feather name="shopping-bag" size={48} color={colors.textSecondary} />
          <Text style={[styles.emptyText, { color: colors.textSecondary }]}>
            No storefronts yet
          </Text>
          <Text style={[styles.emptySubtext, { color: colors.textSecondary }]}>
            Upgrade to Pro or Business to open your storefront
          </Text>
        </View>
      )}

      {!loading && !error && stores.length > 0 && (
        <FlatList
          data={stores}
          keyExtractor={(item) => item.id}
          numColumns={2}
          columnWrapperStyle={styles.row}
          contentContainerStyle={styles.list}
          showsVerticalScrollIndicator={false}
          renderItem={({ item }) => <StoreCard store={item} colors={colors} />}
        />
      )}
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
  title: { fontSize: 17, fontFamily: "Inter_700Bold" },
  centered: { flex: 1, alignItems: "center", justifyContent: "center", gap: 12, padding: 32 },
  emptyText: { fontSize: 16, fontFamily: "Inter_600SemiBold", textAlign: "center" },
  emptySubtext: { fontSize: 13, textAlign: "center", lineHeight: 20 },
  list: { paddingHorizontal: 12, paddingBottom: 32 },
  row: { gap: 12, marginBottom: 12 },
  card: {
    flex: 1,
    borderRadius: 16,
    borderWidth: 1,
    overflow: "hidden",
  },
  cardBanner: { height: 56 },
  cardBody: { padding: 12, gap: 4 },
  avatar: {
    width: 44,
    height: 44,
    borderRadius: 10,
    borderWidth: 3,
    borderColor: "#fff",
    alignItems: "center",
    justifyContent: "center",
    marginTop: -24,
    marginBottom: 4,
  },
  avatarText: { fontSize: 18, fontFamily: "Inter_700Bold", color: "#1A1A1A" },
  storeName: { fontSize: 13, fontFamily: "Inter_700Bold" },
  storeDesc: { fontSize: 11, lineHeight: 16 },
  viewsRow: { flexDirection: "row", alignItems: "center", gap: 3, marginTop: 4 },
  viewsText: { fontSize: 10 },
});
