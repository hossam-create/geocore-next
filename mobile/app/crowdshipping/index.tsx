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

type MenuItem = {
  title: string;
  subtitle: string;
  icon: keyof typeof Feather.glyphMap;
  route: string;
  color: string;
};

const SHOPPER_ITEMS: MenuItem[] = [
  {
    title: "Browse Trips",
    subtitle: "Find travelers going to your route",
    icon: "map",
    route: "/crowdshipping/trips",
    color: "#0071CE",
  },
  {
    title: "Post a Delivery Request",
    subtitle: "Ask travelers to bring an item for you",
    icon: "package",
    route: "/crowdshipping/requests/new",
    color: "#10B981",
  },
  {
    title: "My Delivery Requests",
    subtitle: "Items you're asking travelers to bring",
    icon: "inbox",
    route: "/crowdshipping/requests/my",
    color: "#F59E0B",
  },
];

const TRAVELER_ITEMS: MenuItem[] = [
  {
    title: "Post a Trip",
    subtitle: "Announce your travel to earn rewards",
    icon: "send",
    route: "/crowdshipping/trips/new",
    color: "#8B5CF6",
  },
  {
    title: "My Trips",
    subtitle: "Trips you've posted",
    icon: "compass",
    route: "/crowdshipping/trips/my",
    color: "#0071CE",
  },
  {
    title: "Browse Delivery Requests",
    subtitle: "Find items to deliver on your routes",
    icon: "shopping-bag",
    route: "/crowdshipping/requests",
    color: "#E53935",
  },
];

export default function CrowdshippingHubScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const insets = useSafeAreaInsets();
  const isWeb = Platform.OS === "web";
  const topPad = isWeb ? 67 : insets.top;

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
          Crowdshipping
        </Text>
      </View>

      <ScrollView contentContainerStyle={styles.content} showsVerticalScrollIndicator={false}>
        <View style={styles.hero}>
          <Text style={[styles.heroTitle, { color: colors.text }]}>
            Connect shoppers with travelers
          </Text>
          <Text style={[styles.heroSub, { color: colors.textTertiary }]}>
            Save on shipping by matching items with travelers already going there.
          </Text>
        </View>

        <Section title="I'm a Shopper" subtitle="I want an item delivered" items={SHOPPER_ITEMS} colors={colors} />

        <View style={{ height: 8 }} />

        <Section title="I'm a Traveler" subtitle="I want to earn by delivering" items={TRAVELER_ITEMS} colors={colors} />

        <View style={{ height: 32 }} />
      </ScrollView>
    </View>
  );
}

function Section({
  title,
  subtitle,
  items,
  colors,
}: {
  title: string;
  subtitle: string;
  items: MenuItem[];
  colors: (typeof Colors)["light"];
}) {
  return (
    <View style={styles.section}>
      <Text style={[styles.sectionTitle, { color: colors.text }]}>{title}</Text>
      <Text style={[styles.sectionSub, { color: colors.textTertiary }]}>{subtitle}</Text>
      <View
        style={[
          styles.menuCard,
          { backgroundColor: colors.backgroundSecondary, borderColor: colors.border },
        ]}
      >
        {items.map((item, idx) => (
          <Pressable
            key={item.title}
            onPress={() => router.push(item.route as never)}
            style={({ pressed }) => [
              styles.menuItem,
              idx < items.length - 1 && {
                borderBottomWidth: 1,
                borderBottomColor: colors.borderLight,
              },
              pressed && { opacity: 0.7 },
            ]}
          >
            <View style={[styles.menuIcon, { backgroundColor: item.color + "22" }]}>
              <Feather name={item.icon} size={18} color={item.color} />
            </View>
            <View style={{ flex: 1 }}>
              <Text style={[styles.menuTitle, { color: colors.text }]}>{item.title}</Text>
              <Text style={[styles.menuSub, { color: colors.textTertiary }]}>
                {item.subtitle}
              </Text>
            </View>
            <Feather name="chevron-right" size={18} color={colors.textTertiary} />
          </Pressable>
        ))}
      </View>
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
  headerTitle: { flex: 1, fontSize: 20, fontFamily: "Inter_700Bold" },
  content: { paddingHorizontal: 16, paddingTop: 20 },
  hero: { marginBottom: 20 },
  heroTitle: { fontSize: 22, fontFamily: "Inter_700Bold", marginBottom: 6 },
  heroSub: { fontSize: 14, fontFamily: "Inter_400Regular", lineHeight: 20 },
  section: { marginTop: 12 },
  sectionTitle: { fontSize: 16, fontFamily: "Inter_600SemiBold", marginBottom: 2 },
  sectionSub: { fontSize: 12, fontFamily: "Inter_400Regular", marginBottom: 10 },
  menuCard: {
    borderRadius: 14,
    borderWidth: 1,
    overflow: "hidden",
  },
  menuItem: {
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: 14,
    paddingVertical: 14,
    gap: 12,
  },
  menuIcon: {
    width: 36,
    height: 36,
    borderRadius: 10,
    alignItems: "center",
    justifyContent: "center",
  },
  menuTitle: { fontSize: 15, fontFamily: "Inter_600SemiBold", marginBottom: 2 },
  menuSub: { fontSize: 12, fontFamily: "Inter_400Regular" },
});
