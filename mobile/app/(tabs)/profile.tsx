import { Feather } from "@expo/vector-icons";
import * as Haptics from "expo-haptics";
import { router } from "expo-router";
import React from "react";
import {
  Alert,
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
import { useAppContext } from "@/context/AppContext";
import { useAuthStore } from "@/store/authStore";
import { formatPrice } from "@/utils/format";

const MENU_ITEMS = [
  { id: "listings", label: "My Listings", icon: "list", route: "/my-listings" },
  { id: "favorites", label: "Favorites", icon: "heart", route: "/favorites" },
  { id: "bids", label: "My Bids", icon: "zap", route: "/my-bids" },
  { id: "crowdshipping", label: "Crowdshipping", icon: "truck", route: "/crowdshipping" },
  { id: "balance", label: "Wallet & Payments", icon: "credit-card", route: "/wallet" },
  { id: "settings", label: "Settings", icon: "settings", route: "/settings" },
  { id: "help", label: "Help & Support", icon: "help-circle", route: "/help" },
];

export default function ProfileScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const insets = useSafeAreaInsets();
  const isWeb = Platform.OS === "web";
  const topPad = isWeb ? 67 : insets.top;
  const { user: mockUser, listings, favorites } = useAppContext();
  const { user: authUser, isAuthenticated, logout } = useAuthStore();

  const user = authUser
    ? {
        ...mockUser,
        id: authUser.id,
        name: authUser.name,
        email: authUser.email,
        location: authUser.location ?? mockUser.location,
        isVerified: authUser.isVerified ?? false,
        rating: authUser.rating ?? mockUser.rating,
        totalSales: authUser.totalSales ?? mockUser.totalSales,
        balance: authUser.balance ?? mockUser.balance,
      }
    : mockUser;

  const handleLogout = () => {
    Alert.alert("Sign Out", "Are you sure you want to sign out?", [
      { text: "Cancel", style: "cancel" },
      {
        text: "Sign Out",
        style: "destructive",
        onPress: async () => {
          await logout();
        },
      },
    ]);
  };

  const myListings = listings.filter((l) => l.sellerId === user.id);

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <ScrollView
        contentInsetAdjustmentBehavior="automatic"
        showsVerticalScrollIndicator={false}
      >
        <View
          style={[
            styles.profileHeader,
            {
              paddingTop: topPad + 20,
              backgroundColor: "#0071CE",
            },
          ]}
        >
          <View style={styles.avatarSection}>
            <View style={[styles.avatar, { backgroundColor: colors.tint }]}>
              <Feather name="user" size={32} color="#fff" />
            </View>
            {user.isVerified && (
              <View style={[styles.verifiedBadge, { backgroundColor: "#10B981" }]}>
                <Feather name="check" size={10} color="#fff" />
              </View>
            )}
          </View>

          <Text style={styles.userName}>{user.name}</Text>
          <View style={styles.locationRow}>
            <Feather name="map-pin" size={12} color="rgba(255,255,255,0.6)" />
            <Text style={styles.locationText}>{user.location}</Text>
          </View>

          <View style={styles.statsRow}>
            <View style={styles.stat}>
              <Text style={styles.statValue}>{myListings.length}</Text>
              <Text style={styles.statLabel}>Listings</Text>
            </View>
            <View style={[styles.statDivider, { backgroundColor: "rgba(255,255,255,0.2)" }]} />
            <View style={styles.stat}>
              <Text style={styles.statValue}>{user.totalSales}</Text>
              <Text style={styles.statLabel}>Sold</Text>
            </View>
            <View style={[styles.statDivider, { backgroundColor: "rgba(255,255,255,0.2)" }]} />
            <View style={styles.stat}>
              <Text style={styles.statValue}>{user.rating}</Text>
              <Text style={styles.statLabel}>Rating</Text>
            </View>
          </View>

          <View style={styles.balanceCard}>
            <View>
              <Text style={styles.balanceLabel}>Available Balance</Text>
              <Text style={styles.balanceAmount}>{formatPrice(user.balance)}</Text>
            </View>
            <Pressable
              onPress={() => {
                Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
                router.push("/wallet");
              }}
              style={({ pressed }) => [
                styles.withdrawBtn,
                pressed && { opacity: 0.8 },
              ]}
            >
              <Text style={styles.withdrawText}>Withdraw</Text>
            </Pressable>
          </View>
        </View>

        <View style={[styles.menuCard, { backgroundColor: colors.backgroundSecondary, borderColor: colors.border }]}>
          {MENU_ITEMS.map((item, index) => (
            <React.Fragment key={item.id}>
              <Pressable
                onPress={() => {
                  Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
                  router.push(item.route as any);
                }}
                style={({ pressed }) => [
                  styles.menuRow,
                  pressed && { backgroundColor: colors.backgroundTertiary },
                ]}
              >
                <View style={[styles.menuIcon, { backgroundColor: colors.backgroundTertiary }]}>
                  <Feather name={item.icon as any} size={16} color={colors.tint} />
                </View>
                <Text style={[styles.menuLabel, { color: colors.text }]}>
                  {item.label}
                </Text>
                {item.id === "favorites" && favorites.length > 0 && (
                  <View style={[styles.menuBadge, { backgroundColor: colors.tint }]}>
                    <Text style={styles.menuBadgeText}>{favorites.length}</Text>
                  </View>
                )}
                <Feather name="chevron-right" size={16} color={colors.textTertiary} style={styles.chevron} />
              </Pressable>
              {index < MENU_ITEMS.length - 1 && (
                <View style={[styles.divider, { backgroundColor: colors.border }]} />
              )}
            </React.Fragment>
          ))}
        </View>

        <Text style={[styles.memberSince, { color: colors.textTertiary }]}>
          Member since {new Date(user.joinedAt).toLocaleDateString("en-AE", { month: "long", year: "numeric" })}
        </Text>

        {isAuthenticated ? (
          <Pressable
            onPress={handleLogout}
            style={({ pressed }) => [
              styles.authBtn,
              {
                borderColor: colors.error,
                opacity: pressed ? 0.8 : 1,
              },
            ]}
          >
            <Feather name="log-out" size={16} color={colors.error} />
            <Text style={[styles.authBtnText, { color: colors.error }]}>
              Sign Out
            </Text>
          </Pressable>
        ) : (
          <Pressable
            onPress={() => router.push("/login")}
            style={({ pressed }) => [
              styles.authBtn,
              {
                backgroundColor: "#0071CE",
                borderColor: "#0071CE",
                opacity: pressed ? 0.85 : 1,
              },
            ]}
          >
            <Feather name="log-in" size={16} color="#fff" />
            <Text style={[styles.authBtnText, { color: "#fff" }]}>
              Sign In / Register
            </Text>
          </Pressable>
        )}

        <View style={{ height: isWeb ? 34 : 100 }} />
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  profileHeader: {
    alignItems: "center",
    paddingHorizontal: 24,
    paddingBottom: 24,
  },
  avatarSection: {
    position: "relative",
    marginBottom: 12,
  },
  avatar: {
    width: 80,
    height: 80,
    borderRadius: 40,
    alignItems: "center",
    justifyContent: "center",
  },
  verifiedBadge: {
    position: "absolute",
    bottom: 0,
    right: 0,
    width: 22,
    height: 22,
    borderRadius: 11,
    alignItems: "center",
    justifyContent: "center",
    borderWidth: 2,
    borderColor: "#005BA1",
  },
  userName: {
    fontSize: 22,
    fontFamily: "Inter_700Bold",
    color: "#fff",
  },
  locationRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 4,
    marginTop: 4,
    marginBottom: 16,
  },
  locationText: {
    fontSize: 13,
    fontFamily: "Inter_400Regular",
    color: "rgba(255,255,255,0.6)",
  },
  statsRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 24,
    marginBottom: 20,
  },
  stat: {
    alignItems: "center",
  },
  statValue: {
    fontSize: 20,
    fontFamily: "Inter_700Bold",
    color: "#fff",
  },
  statLabel: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
    color: "rgba(255,255,255,0.6)",
    marginTop: 2,
  },
  statDivider: {
    width: 1,
    height: 30,
  },
  balanceCard: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    backgroundColor: "rgba(255,255,255,0.1)",
    borderRadius: 14,
    padding: 16,
    width: "100%",
  },
  balanceLabel: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
    color: "rgba(255,255,255,0.6)",
  },
  balanceAmount: {
    fontSize: 20,
    fontFamily: "Inter_700Bold",
    color: "#fff",
    marginTop: 2,
  },
  withdrawBtn: {
    backgroundColor: "#FFC220",
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderRadius: 8,
  },
  withdrawText: {
    fontSize: 13,
    fontFamily: "Inter_600SemiBold",
    color: "#1A1A1A",
  },
  menuCard: {
    margin: 16,
    borderRadius: 16,
    borderWidth: 1,
    overflow: "hidden",
  },
  menuRow: {
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: 16,
    paddingVertical: 14,
    gap: 12,
  },
  menuIcon: {
    width: 34,
    height: 34,
    borderRadius: 10,
    alignItems: "center",
    justifyContent: "center",
  },
  menuLabel: {
    flex: 1,
    fontSize: 15,
    fontFamily: "Inter_500Medium",
  },
  menuBadge: {
    minWidth: 20,
    height: 20,
    borderRadius: 10,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 5,
  },
  menuBadgeText: {
    fontSize: 11,
    fontFamily: "Inter_700Bold",
    color: "#fff",
  },
  chevron: {
    marginLeft: "auto",
  },
  divider: {
    height: 1,
    marginLeft: 62,
  },
  memberSince: {
    textAlign: "center",
    fontSize: 13,
    fontFamily: "Inter_400Regular",
    marginBottom: 8,
  },
  authBtn: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: 8,
    marginHorizontal: 16,
    marginBottom: 12,
    paddingVertical: 14,
    borderRadius: 12,
    borderWidth: 1.5,
  },
  authBtnText: {
    fontSize: 15,
    fontFamily: "Inter_600SemiBold",
  },
});
