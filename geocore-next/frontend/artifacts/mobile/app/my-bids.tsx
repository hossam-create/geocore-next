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

import Colors from "@/constants/colors";
import { useAppContext } from "@/context/AppContext";
import { formatPrice, getAuctionTimeLeft } from "@/utils/format";

export default function MyBidsScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const insets = useSafeAreaInsets();
  const isWeb = Platform.OS === "web";
  const topPad = isWeb ? 67 : insets.top;
  const { listings } = useAppContext();

  const auctionListings = listings.filter((l) => l.isAuction);

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
          Active Auctions
        </Text>
      </View>

      <FlatList
        data={auctionListings}
        keyExtractor={(item) => item.id}
        contentContainerStyle={styles.list}
        contentInsetAdjustmentBehavior="automatic"
        showsVerticalScrollIndicator={false}
        renderItem={({ item }) => {
          const timeLeft = item.auctionEndsAt
            ? getAuctionTimeLeft(item.auctionEndsAt)
            : null;
          const isEnded = timeLeft === "Ended";

          return (
            <Pressable
              onPress={() =>
                router.push({ pathname: "/listing/[id]", params: { id: item.id } })
              }
              style={({ pressed }) => [
                styles.auctionRow,
                {
                  backgroundColor: colors.backgroundSecondary,
                  borderColor: colors.border,
                  opacity: pressed ? 0.85 : 1,
                },
              ]}
            >
              <View
                style={[
                  styles.auctionImage,
                  { backgroundColor: colors.backgroundTertiary },
                ]}
              >
                <Feather name="image" size={20} color={colors.textTertiary} />
              </View>
              <View style={styles.auctionInfo}>
                <Text
                  style={[styles.auctionTitle, { color: colors.text }]}
                  numberOfLines={2}
                >
                  {item.title}
                </Text>
                <View style={styles.auctionMeta}>
                  <Text style={[styles.bidCount, { color: colors.textTertiary }]}>
                    {item.bidCount} bids
                  </Text>
                  {timeLeft && (
                    <View
                      style={[
                        styles.timer,
                        {
                          backgroundColor: isEnded
                            ? colors.backgroundTertiary
                            : colors.tint + "15",
                        },
                      ]}
                    >
                      <Feather
                        name="clock"
                        size={11}
                        color={isEnded ? colors.textTertiary : colors.tint}
                      />
                      <Text
                        style={[
                          styles.timerText,
                          {
                            color: isEnded ? colors.textTertiary : colors.tint,
                          },
                        ]}
                      >
                        {timeLeft}
                      </Text>
                    </View>
                  )}
                </View>
                <Text style={[styles.currentBid, { color: colors.tint }]}>
                  {formatPrice(item.currentBid ?? 0)}
                </Text>
              </View>
              <Feather name="chevron-right" size={18} color={colors.textTertiary} />
            </Pressable>
          );
        }}
        ListEmptyComponent={
          <View style={styles.empty}>
            <Feather name="zap" size={40} color={colors.textTertiary} />
            <Text style={[styles.emptyTitle, { color: colors.text }]}>
              No active auctions
            </Text>
            <Text style={[styles.emptyText, { color: colors.textTertiary }]}>
              Discover auctions and place bids
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
  list: {
    padding: 16,
    gap: 12,
  },
  auctionRow: {
    flexDirection: "row",
    alignItems: "center",
    padding: 12,
    borderRadius: 14,
    borderWidth: 1,
    gap: 12,
  },
  auctionImage: {
    width: 64,
    height: 64,
    borderRadius: 10,
    alignItems: "center",
    justifyContent: "center",
    flexShrink: 0,
  },
  auctionInfo: {
    flex: 1,
    gap: 4,
  },
  auctionTitle: {
    fontSize: 14,
    fontFamily: "Inter_600SemiBold",
    lineHeight: 20,
  },
  auctionMeta: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
  },
  bidCount: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
  },
  timer: {
    flexDirection: "row",
    alignItems: "center",
    gap: 3,
    paddingHorizontal: 7,
    paddingVertical: 3,
    borderRadius: 6,
  },
  timerText: {
    fontSize: 11,
    fontFamily: "Inter_600SemiBold",
  },
  currentBid: {
    fontSize: 16,
    fontFamily: "Inter_700Bold",
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
});
