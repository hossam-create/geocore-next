import { Feather } from "@expo/vector-icons";
import { router } from "expo-router";
import React, { useState } from "react";
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

const FAQS = [
  {
    id: "f1",
    q: "How do auctions work?",
    a: "Standard auctions let buyers place bids. The highest bidder when the timer ends wins. Dutch auctions start high and decrease. Reverse auctions start at zero and go up.",
  },
  {
    id: "f2",
    q: "How do I get paid?",
    a: "Once a buyer confirms receipt, funds are released to your GeoCore wallet. You can withdraw to your bank account anytime with no fees.",
  },
  {
    id: "f3",
    q: "How do I report a listing?",
    a: "Open the listing and tap the share icon, then select 'Report'. Our moderation team reviews reports within 24 hours.",
  },
  {
    id: "f4",
    q: "What are the selling fees?",
    a: "Listing is free. We charge a 2.5% success fee when your item sells. Featured promotions are available starting at AED 50.",
  },
];

export default function HelpScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const insets = useSafeAreaInsets();
  const isWeb = Platform.OS === "web";
  const topPad = isWeb ? 67 : insets.top;
  const [expanded, setExpanded] = useState<string | null>(null);

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
        <Text style={[styles.headerTitle, { color: colors.text }]}>Help & Support</Text>
      </View>

      <ScrollView
        contentInsetAdjustmentBehavior="automatic"
        showsVerticalScrollIndicator={false}
        contentContainerStyle={styles.content}
      >
        <View style={[styles.contactCard, { backgroundColor: colors.navy }]}>
          <Feather name="headphones" size={28} color={colors.tint} />
          <Text style={styles.contactTitle}>Need help?</Text>
          <Text style={styles.contactSub}>
            Our support team is available 24/7 in Arabic and English
          </Text>
          <Pressable
            style={({ pressed }) => [
              styles.contactBtn,
              { backgroundColor: colors.tint, opacity: pressed ? 0.8 : 1 },
            ]}
          >
            <Feather name="message-square" size={16} color="#fff" />
            <Text style={styles.contactBtnText}>Chat with Support</Text>
          </Pressable>
        </View>

        <Text style={[styles.sectionTitle, { color: colors.text }]}>
          Frequently Asked Questions
        </Text>

        {FAQS.map((faq) => (
          <Pressable
            key={faq.id}
            onPress={() => setExpanded(expanded === faq.id ? null : faq.id)}
            style={[
              styles.faqCard,
              { backgroundColor: colors.backgroundSecondary, borderColor: colors.border },
            ]}
          >
            <View style={styles.faqHeader}>
              <Text style={[styles.faqQ, { color: colors.text }]}>{faq.q}</Text>
              <Feather
                name={expanded === faq.id ? "chevron-up" : "chevron-down"}
                size={18}
                color={colors.textTertiary}
              />
            </View>
            {expanded === faq.id && (
              <Text style={[styles.faqA, { color: colors.textSecondary }]}>
                {faq.a}
              </Text>
            )}
          </Pressable>
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
    flex: 1,
    fontSize: 20,
    fontFamily: "Inter_700Bold",
  },
  content: {
    padding: 16,
    gap: 16,
  },
  contactCard: {
    borderRadius: 16,
    padding: 20,
    alignItems: "center",
    gap: 10,
  },
  contactTitle: {
    fontSize: 20,
    fontFamily: "Inter_700Bold",
    color: "#fff",
  },
  contactSub: {
    fontSize: 13,
    fontFamily: "Inter_400Regular",
    color: "rgba(255,255,255,0.65)",
    textAlign: "center",
    lineHeight: 20,
  },
  contactBtn: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
    paddingHorizontal: 20,
    paddingVertical: 11,
    borderRadius: 10,
    marginTop: 4,
  },
  contactBtnText: {
    fontSize: 14,
    fontFamily: "Inter_600SemiBold",
    color: "#fff",
  },
  sectionTitle: {
    fontSize: 18,
    fontFamily: "Inter_700Bold",
  },
  faqCard: {
    borderRadius: 12,
    borderWidth: 1,
    padding: 16,
    gap: 10,
  },
  faqHeader: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 12,
  },
  faqQ: {
    flex: 1,
    fontSize: 15,
    fontFamily: "Inter_600SemiBold",
    lineHeight: 22,
  },
  faqA: {
    fontSize: 14,
    fontFamily: "Inter_400Regular",
    lineHeight: 22,
  },
});
