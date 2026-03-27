import { Feather } from "@expo/vector-icons";
import { router } from "expo-router";
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
import { useAppContext } from "@/context/AppContext";
import { formatPrice } from "@/utils/format";

const TRANSACTIONS = [
  { id: "t1", type: "credit", label: "Sale: iPhone 13 Pro", amount: 3200, date: "2026-03-15" },
  { id: "t2", type: "debit", label: "Listing fee", amount: -50, date: "2026-03-12" },
  { id: "t3", type: "credit", label: "Sale: Nike Air Max", amount: 450, date: "2026-03-08" },
  { id: "t4", type: "debit", label: "Featured promotion", amount: -150, date: "2026-03-05" },
  { id: "t5", type: "credit", label: "Sale: Sony Headphones", amount: 800, date: "2026-02-28" },
];

export default function WalletScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const insets = useSafeAreaInsets();
  const isWeb = Platform.OS === "web";
  const topPad = isWeb ? 67 : insets.top;
  const { user } = useAppContext();

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <View
        style={[
          styles.header,
          {
            paddingTop: topPad + 12,
            backgroundColor: "#0071CE",
          },
        ]}
      >
        <Pressable
          onPress={() => router.back()}
          style={({ pressed }) => [styles.backBtn, pressed && { opacity: 0.7 }]}
        >
          <Feather name="arrow-left" size={22} color="#fff" />
        </Pressable>
        <Text style={styles.headerTitle}>Wallet</Text>

        <View style={styles.balanceSection}>
          <Text style={styles.balanceLabel}>Available Balance</Text>
          <Text style={styles.balanceAmount}>{formatPrice(user.balance)}</Text>

          <View style={styles.walletActions}>
            <Pressable
              style={({ pressed }) => [
                styles.walletBtn,
                { backgroundColor: colors.tint, opacity: pressed ? 0.8 : 1 },
              ]}
            >
              <Feather name="arrow-up" size={16} color="#fff" />
              <Text style={styles.walletBtnText}>Withdraw</Text>
            </Pressable>
            <Pressable
              style={({ pressed }) => [
                styles.walletBtn,
                { backgroundColor: "rgba(255,255,255,0.15)", opacity: pressed ? 0.8 : 1 },
              ]}
            >
              <Feather name="arrow-down" size={16} color="#fff" />
              <Text style={styles.walletBtnText}>Add Funds</Text>
            </Pressable>
          </View>
        </View>
      </View>

      <ScrollView
        contentInsetAdjustmentBehavior="automatic"
        showsVerticalScrollIndicator={false}
        contentContainerStyle={styles.content}
      >
        <Text style={[styles.sectionTitle, { color: colors.text }]}>
          Transaction History
        </Text>

        <View style={[styles.transactionCard, { backgroundColor: colors.backgroundSecondary, borderColor: colors.border }]}>
          {TRANSACTIONS.map((tx, index) => (
            <React.Fragment key={tx.id}>
              <View style={styles.txRow}>
                <View style={[
                  styles.txIcon,
                  { backgroundColor: tx.amount > 0 ? "#10B98120" : "#EF444420" }
                ]}>
                  <Feather
                    name={tx.amount > 0 ? "arrow-down-left" : "arrow-up-right"}
                    size={16}
                    color={tx.amount > 0 ? "#10B981" : "#EF4444"}
                  />
                </View>
                <View style={styles.txInfo}>
                  <Text style={[styles.txLabel, { color: colors.text }]}>
                    {tx.label}
                  </Text>
                  <Text style={[styles.txDate, { color: colors.textTertiary }]}>
                    {new Date(tx.date).toLocaleDateString("en-AE", {
                      day: "numeric", month: "short", year: "numeric",
                    })}
                  </Text>
                </View>
                <Text style={[
                  styles.txAmount,
                  { color: tx.amount > 0 ? "#10B981" : "#EF4444" }
                ]}>
                  {tx.amount > 0 ? "+" : ""}{formatPrice(Math.abs(tx.amount))}
                </Text>
              </View>
              {index < TRANSACTIONS.length - 1 && (
                <View style={[styles.divider, { backgroundColor: colors.border }]} />
              )}
            </React.Fragment>
          ))}
        </View>

        <View style={{ height: isWeb ? 34 : 100 }} />
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  header: {
    paddingHorizontal: 16,
    paddingBottom: 24,
  },
  backBtn: {
    width: 36,
    height: 36,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: 16,
  },
  headerTitle: {
    fontSize: 16,
    fontFamily: "Inter_600SemiBold",
    color: "rgba(255,255,255,0.7)",
    marginBottom: 8,
  },
  balanceSection: {
    gap: 8,
  },
  balanceLabel: {
    fontSize: 13,
    fontFamily: "Inter_400Regular",
    color: "rgba(255,255,255,0.6)",
  },
  balanceAmount: {
    fontSize: 36,
    fontFamily: "Inter_700Bold",
    color: "#fff",
  },
  walletActions: {
    flexDirection: "row",
    gap: 10,
    marginTop: 8,
  },
  walletBtn: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
    paddingHorizontal: 18,
    paddingVertical: 10,
    borderRadius: 10,
  },
  walletBtnText: {
    fontSize: 14,
    fontFamily: "Inter_600SemiBold",
    color: "#fff",
  },
  content: {
    padding: 16,
    gap: 16,
  },
  sectionTitle: {
    fontSize: 18,
    fontFamily: "Inter_700Bold",
  },
  transactionCard: {
    borderRadius: 16,
    borderWidth: 1,
    overflow: "hidden",
  },
  txRow: {
    flexDirection: "row",
    alignItems: "center",
    padding: 14,
    gap: 12,
  },
  txIcon: {
    width: 38,
    height: 38,
    borderRadius: 12,
    alignItems: "center",
    justifyContent: "center",
  },
  txInfo: {
    flex: 1,
    gap: 2,
  },
  txLabel: {
    fontSize: 14,
    fontFamily: "Inter_500Medium",
  },
  txDate: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
  },
  txAmount: {
    fontSize: 15,
    fontFamily: "Inter_700Bold",
  },
  divider: {
    height: 1,
    marginLeft: 64,
  },
});
