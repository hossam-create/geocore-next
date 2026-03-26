import { Feather } from "@expo/vector-icons";
import { router } from "expo-router";
import React from "react";
import {
  FlatList,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  View,
  useColorScheme,
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import ListingCard from "@/components/ListingCard";
import Colors from "@/constants/colors";
import { useAppContext } from "@/context/AppContext";

export default function MyListingsScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const insets = useSafeAreaInsets();
  const isWeb = Platform.OS === "web";
  const topPad = isWeb ? 67 : insets.top;
  const { listings, user } = useAppContext();

  const myListings = listings.filter((l) => l.sellerId === user.id);

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
          style={({ pressed }) => [styles.backBtn, pressed && { opacity: 0.7 }]}
        >
          <Feather name="arrow-left" size={22} color={colors.text} />
        </Pressable>
        <Text style={[styles.headerTitle, { color: colors.text }]}>
          My Listings
        </Text>
        <Pressable
          onPress={() => router.push("/(tabs)/sell")}
          style={({ pressed }) => [styles.addBtn, { backgroundColor: colors.tint, opacity: pressed ? 0.8 : 1 }]}
        >
          <Feather name="plus" size={18} color="#fff" />
        </Pressable>
      </View>

      <FlatList
        data={myListings}
        keyExtractor={(item) => item.id}
        contentContainerStyle={styles.list}
        contentInsetAdjustmentBehavior="automatic"
        showsVerticalScrollIndicator={false}
        renderItem={({ item }) => <ListingCard listing={item} />}
        ListEmptyComponent={
          <View style={styles.empty}>
            <Feather name="package" size={40} color={colors.textTertiary} />
            <Text style={[styles.emptyTitle, { color: colors.text }]}>
              No listings yet
            </Text>
            <Text style={[styles.emptyText, { color: colors.textTertiary }]}>
              Start selling by tapping the + button
            </Text>
            <Pressable
              onPress={() => router.push("/(tabs)/sell")}
              style={[styles.sellBtn, { backgroundColor: colors.tint }]}
            >
              <Text style={styles.sellBtnText}>Post a Listing</Text>
            </Pressable>
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
    flex: 1,
    fontSize: 20,
    fontFamily: "Inter_700Bold",
  },
  addBtn: {
    width: 36,
    height: 36,
    borderRadius: 18,
    alignItems: "center",
    justifyContent: "center",
  },
  list: {
    paddingHorizontal: 16,
    paddingTop: 16,
  },
  empty: {
    alignItems: "center",
    justifyContent: "center",
    paddingTop: 80,
    gap: 12,
    paddingHorizontal: 40,
  },
  emptyTitle: {
    fontSize: 18,
    fontFamily: "Inter_600SemiBold",
  },
  emptyText: {
    fontSize: 14,
    fontFamily: "Inter_400Regular",
    textAlign: "center",
  },
  sellBtn: {
    marginTop: 8,
    paddingHorizontal: 24,
    paddingVertical: 12,
    borderRadius: 12,
  },
  sellBtnText: {
    fontSize: 15,
    fontFamily: "Inter_600SemiBold",
    color: "#fff",
  },
});
