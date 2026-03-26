import { Feather } from "@expo/vector-icons";
import * as Haptics from "expo-haptics";
import { router } from "expo-router";
import React, { useEffect, useRef, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  Modal,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Switch,
  Text,
  TextInput,
  View,
  useColorScheme,
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import Colors from "@/constants/colors";
import {
  type ListingCategory,
  type ListingCondition,
  useAppContext,
} from "@/context/AppContext";
import { GCC_CURRENCIES } from "@/utils/currency";
import { geocodeAddress } from "@/utils/geo";
import {
  getUpcomingHolidays,
  inferCountryCode,
} from "@/utils/holidays";
import { validateEmailFormat, validatePhoneE164 } from "@/utils/validation";

const CATEGORIES: { id: ListingCategory; label: string; icon: string }[] = [
  { id: "vehicles", label: "Vehicles", icon: "truck" },
  { id: "electronics", label: "Electronics", icon: "cpu" },
  { id: "real-estate", label: "Real Estate", icon: "home" },
  { id: "fashion", label: "Fashion", icon: "shopping-bag" },
  { id: "furniture", label: "Furniture", icon: "box" },
  { id: "sports", label: "Sports", icon: "activity" },
  { id: "services", label: "Services", icon: "briefcase" },
];

const CONDITIONS: { id: ListingCondition; label: string }[] = [
  { id: "new", label: "Brand New" },
  { id: "like-new", label: "Like New" },
  { id: "good", label: "Good" },
  { id: "fair", label: "Fair" },
  { id: "poor", label: "For Parts" },
];

export default function SellScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const insets = useSafeAreaInsets();
  const isWeb = Platform.OS === "web";
  const topPad = isWeb ? 67 : insets.top;
  const { addListing, user, detectedLocation, activeCurrency, setActiveCurrency } =
    useAppContext();

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [price, setPrice] = useState("");
  const [currency, setCurrency] = useState(activeCurrency);
  const [location, setLocation] = useState(detectedLocation || user.location);
  const [category, setCategory] = useState<ListingCategory>("electronics");
  const [condition, setCondition] = useState<ListingCondition>("good");
  const [isAuction, setIsAuction] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showCurrencyPicker, setShowCurrencyPicker] = useState(false);

  const [geocoding, setGeocoding] = useState(false);
  const [geocoded, setGeocoded] = useState<{
    lat: number;
    lon: number;
  } | null>(null);

  const [contactEmail, setContactEmail] = useState(user.email);
  const [contactPhone, setContactPhone] = useState(user.phone ?? "");
  const [emailError, setEmailError] = useState("");
  const [phoneError, setPhoneError] = useState("");

  const [holidayWarning, setHolidayWarning] = useState<string | null>(null);

  const geocodeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (detectedLocation) setLocation(detectedLocation);
  }, [detectedLocation]);

  useEffect(() => {
    if (geocodeTimer.current) clearTimeout(geocodeTimer.current);
    if (!location.trim()) { setGeocoded(null); return; }
    geocodeTimer.current = setTimeout(async () => {
      setGeocoding(true);
      const result = await geocodeAddress(location);
      setGeocoding(false);
      if (result) {
        setGeocoded({ lat: parseFloat(result.lat), lon: parseFloat(result.lon) });
      } else {
        setGeocoded(null);
      }
    }, 800);
    return () => {
      if (geocodeTimer.current) clearTimeout(geocodeTimer.current);
    };
  }, [location]);

  useEffect(() => {
    if (!isAuction) { setHolidayWarning(null); return; }
    const code = inferCountryCode(location);
    getUpcomingHolidays(code, 7).then((holidays) => {
      if (holidays.length > 0) {
        setHolidayWarning(
          `⚠️ ${holidays[0].name} (${holidays[0].date}) is upcoming in your region — auctions may have lower activity.`
        );
      } else {
        setHolidayWarning(null);
      }
    });
  }, [isAuction, location]);

  const handleEmailBlur = () => {
    if (contactEmail && !validateEmailFormat(contactEmail)) {
      setEmailError("Enter a valid email address");
    } else {
      setEmailError("");
    }
  };

  const handlePhoneBlur = () => {
    if (contactPhone && !validatePhoneE164(contactPhone)) {
      setPhoneError("Use E.164 format, e.g. +971501234567");
    } else {
      setPhoneError("");
    }
  };

  const handleSubmit = async () => {
    if (!title.trim() || !description.trim() || (!price.trim() && !isAuction)) {
      Alert.alert("Missing fields", "Please fill in all required fields.");
      return;
    }
    if (emailError || phoneError) {
      Alert.alert("Fix errors", "Please correct the contact field errors.");
      return;
    }

    Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
    setIsSubmitting(true);

    try {
      await addListing({
        title: title.trim(),
        description: description.trim(),
        price: isAuction ? 0 : parseFloat(price) || 0,
        currency,
        category,
        condition,
        location: location.trim() || user.location,
        lat: geocoded?.lat,
        lon: geocoded?.lon,
        sellerId: user.id,
        sellerName: user.name,
        isAuction,
        auctionType: isAuction ? "standard" : undefined,
        auctionEndsAt: isAuction
          ? new Date(Date.now() + 86400000 * 3).toISOString()
          : undefined,
        currentBid: isAuction ? 0 : undefined,
        bidCount: isAuction ? 0 : undefined,
        tags: [],
        isFeatured: false,
      });

      Alert.alert(
        "Listing Published!",
        "Your item is now live and visible to buyers.",
        [{ text: "View Listings", onPress: () => router.push("/") }]
      );

      setTitle("");
      setDescription("");
      setPrice("");
      setIsAuction(false);
    } finally {
      setIsSubmitting(false);
    }
  };

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
        <Text style={[styles.headerTitle, { color: colors.text }]}>
          Post a Listing
        </Text>
        <Text style={[styles.headerSubtitle, { color: colors.textTertiary }]}>
          Reach thousands of buyers in the GCC
        </Text>
      </View>

      <ScrollView
        style={styles.scroll}
        contentInsetAdjustmentBehavior="automatic"
        showsVerticalScrollIndicator={false}
        keyboardShouldPersistTaps="handled"
        contentContainerStyle={styles.content}
      >
        <View
          style={[
            styles.uploadArea,
            { backgroundColor: colors.backgroundTertiary, borderColor: colors.border },
          ]}
        >
          <Feather name="camera" size={28} color={colors.textTertiary} />
          <Text style={[styles.uploadText, { color: colors.textTertiary }]}>
            Add Photos
          </Text>
          <Text style={[styles.uploadHint, { color: colors.textTertiary }]}>
            Up to 10 photos
          </Text>
        </View>

        <FormSection label="Title *">
          <TextInput
            style={[
              styles.textInput,
              { color: colors.text, backgroundColor: colors.backgroundSecondary, borderColor: colors.border },
            ]}
            placeholder="What are you selling?"
            placeholderTextColor={colors.textTertiary}
            value={title}
            onChangeText={setTitle}
            maxLength={100}
          />
        </FormSection>

        <FormSection label="Description *">
          <TextInput
            style={[
              styles.textInput,
              styles.textArea,
              { color: colors.text, backgroundColor: colors.backgroundSecondary, borderColor: colors.border },
            ]}
            placeholder="Describe your item in detail — condition, history, reason for selling..."
            placeholderTextColor={colors.textTertiary}
            value={description}
            onChangeText={setDescription}
            multiline
            numberOfLines={4}
            maxLength={1000}
          />
        </FormSection>

        <FormSection label="Category">
          <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.chipRow}>
            {CATEGORIES.map((cat) => (
              <Pressable
                key={cat.id}
                onPress={() => {
                  Haptics.selectionAsync();
                  setCategory(cat.id);
                }}
                style={[
                  styles.chip,
                  {
                    backgroundColor:
                      category === cat.id ? colors.tint : colors.backgroundTertiary,
                  },
                ]}
              >
                <Feather name={cat.icon as any} size={13} color={category === cat.id ? "#fff" : colors.textSecondary} />
                <Text
                  style={[
                    styles.chipLabel,
                    { color: category === cat.id ? "#fff" : colors.textSecondary },
                  ]}
                >
                  {cat.label}
                </Text>
              </Pressable>
            ))}
          </ScrollView>
        </FormSection>

        <FormSection label="Condition">
          <View style={styles.conditionRow}>
            {CONDITIONS.map((c) => (
              <Pressable
                key={c.id}
                onPress={() => {
                  Haptics.selectionAsync();
                  setCondition(c.id);
                }}
                style={[
                  styles.conditionChip,
                  {
                    backgroundColor: condition === c.id ? colors.tint : colors.backgroundTertiary,
                    borderColor: condition === c.id ? colors.tint : colors.border,
                  },
                ]}
              >
                <Text
                  style={[
                    styles.conditionLabel,
                    { color: condition === c.id ? "#fff" : colors.textSecondary },
                  ]}
                >
                  {c.label}
                </Text>
              </Pressable>
            ))}
          </View>
        </FormSection>

        <View
          style={[
            styles.auctionToggle,
            { backgroundColor: colors.backgroundSecondary, borderColor: colors.border },
          ]}
        >
          <View>
            <Text style={[styles.toggleLabel, { color: colors.text }]}>
              List as Auction
            </Text>
            <Text style={[styles.toggleHint, { color: colors.textTertiary }]}>
              Buyers bid on your item
            </Text>
          </View>
          <Switch
            value={isAuction}
            onValueChange={(v) => {
              Haptics.selectionAsync();
              setIsAuction(v);
            }}
            trackColor={{ false: colors.border, true: colors.tint }}
            thumbColor="#fff"
          />
        </View>

        {holidayWarning && (
          <View
            style={[
              styles.warningBanner,
              { backgroundColor: "#FFF7ED", borderColor: "#FB923C" },
            ]}
          >
            <Text style={styles.warningText}>{holidayWarning}</Text>
          </View>
        )}

        {!isAuction && (
          <FormSection label="Price *">
            <View style={styles.priceRow}>
              <Pressable
                onPress={() => setShowCurrencyPicker(true)}
                style={[
                  styles.currencyBtn,
                  { backgroundColor: colors.backgroundSecondary, borderColor: colors.border },
                ]}
              >
                <Text style={[styles.currencyCode, { color: colors.text }]}>
                  {currency}
                </Text>
                <Feather name="chevron-down" size={14} color={colors.textTertiary} />
              </Pressable>
              <TextInput
                style={[
                  styles.textInput,
                  styles.priceInput,
                  { color: colors.text, backgroundColor: colors.backgroundSecondary, borderColor: colors.border },
                ]}
                placeholder="0"
                placeholderTextColor={colors.textTertiary}
                value={price}
                onChangeText={setPrice}
                keyboardType="numeric"
              />
            </View>
          </FormSection>
        )}

        <FormSection label="Location">
          <View style={styles.locationRow}>
            <TextInput
              style={[
                styles.textInput,
                styles.locationInput,
                { color: colors.text, backgroundColor: colors.backgroundSecondary, borderColor: colors.border },
              ]}
              placeholder="City, Country"
              placeholderTextColor={colors.textTertiary}
              value={location}
              onChangeText={setLocation}
            />
            {geocoding && (
              <ActivityIndicator size="small" color={colors.tint} style={styles.geoIndicator} />
            )}
            {geocoded && !geocoding && (
              <Feather name="map-pin" size={16} color={colors.tint} style={styles.geoIndicator} />
            )}
          </View>
          {geocoded && !geocoding && (
            <Text style={[styles.geoHint, { color: colors.tint }]}>
              📍 Geocoded: {geocoded.lat.toFixed(4)}, {geocoded.lon.toFixed(4)}
            </Text>
          )}
        </FormSection>

        <Text style={[styles.sectionTitle, { color: colors.textSecondary }]}>
          CONTACT INFO
        </Text>

        <FormSection label="Email">
          <TextInput
            style={[
              styles.textInput,
              {
                color: colors.text,
                backgroundColor: colors.backgroundSecondary,
                borderColor: emailError ? "#EF4444" : colors.border,
              },
            ]}
            placeholder="your@email.com"
            placeholderTextColor={colors.textTertiary}
            value={contactEmail}
            onChangeText={(v) => { setContactEmail(v); setEmailError(""); }}
            onBlur={handleEmailBlur}
            keyboardType="email-address"
            autoCapitalize="none"
          />
          {emailError ? (
            <Text style={styles.fieldError}>{emailError}</Text>
          ) : null}
        </FormSection>

        <FormSection label="Phone (E.164 format)">
          <TextInput
            style={[
              styles.textInput,
              {
                color: colors.text,
                backgroundColor: colors.backgroundSecondary,
                borderColor: phoneError ? "#EF4444" : colors.border,
              },
            ]}
            placeholder="+971501234567"
            placeholderTextColor={colors.textTertiary}
            value={contactPhone}
            onChangeText={(v) => { setContactPhone(v); setPhoneError(""); }}
            onBlur={handlePhoneBlur}
            keyboardType="phone-pad"
          />
          {phoneError ? (
            <Text style={styles.fieldError}>{phoneError}</Text>
          ) : null}
        </FormSection>

        <Pressable
          onPress={handleSubmit}
          disabled={isSubmitting}
          style={({ pressed }) => [
            styles.submitBtn,
            { backgroundColor: colors.tint, opacity: pressed || isSubmitting ? 0.8 : 1 },
          ]}
        >
          <Feather name="check" size={18} color="#fff" />
          <Text style={styles.submitText}>
            {isSubmitting ? "Publishing..." : "Publish Listing"}
          </Text>
        </Pressable>

        <View style={{ height: isWeb ? 34 : 100 }} />
      </ScrollView>

      <Modal
        visible={showCurrencyPicker}
        animationType="slide"
        presentationStyle="pageSheet"
        onRequestClose={() => setShowCurrencyPicker(false)}
      >
        <View style={[styles.modalContainer, { backgroundColor: colors.background }]}>
          <View style={[styles.modalHeader, { borderBottomColor: colors.border }]}>
            <Text style={[styles.modalTitle, { color: colors.text }]}>
              Select Currency
            </Text>
            <Pressable onPress={() => setShowCurrencyPicker(false)}>
              <Feather name="x" size={22} color={colors.text} />
            </Pressable>
          </View>
          <ScrollView>
            {GCC_CURRENCIES.map((c) => (
              <Pressable
                key={c.code}
                onPress={() => {
                  setCurrency(c.code);
                  setActiveCurrency(c.code);
                  Haptics.selectionAsync();
                  setShowCurrencyPicker(false);
                }}
                style={[
                  styles.currencyOption,
                  {
                    backgroundColor:
                      currency === c.code ? colors.backgroundTertiary : "transparent",
                    borderBottomColor: colors.border,
                  },
                ]}
              >
                <Text style={[styles.currencySymbol, { color: colors.tint }]}>
                  {c.symbol}
                </Text>
                <View>
                  <Text style={[styles.currencyName, { color: colors.text }]}>
                    {c.name}
                  </Text>
                  <Text style={[styles.currencyCodeLabel, { color: colors.textTertiary }]}>
                    {c.code}
                  </Text>
                </View>
                {currency === c.code && (
                  <Feather name="check" size={18} color={colors.tint} style={{ marginLeft: "auto" }} />
                )}
              </Pressable>
            ))}
          </ScrollView>
        </View>
      </Modal>
    </View>
  );
}

function FormSection({ label, children }: { label: string; children: React.ReactNode }) {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  return (
    <View style={styles.formSection}>
      <Text style={[styles.formLabel, { color: colors.textSecondary }]}>{label}</Text>
      {children}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  header: {
    borderBottomWidth: 1,
    paddingHorizontal: 16,
    paddingBottom: 14,
  },
  headerTitle: {
    fontSize: 24,
    fontFamily: "Inter_700Bold",
  },
  headerSubtitle: {
    fontSize: 13,
    fontFamily: "Inter_400Regular",
    marginTop: 2,
  },
  scroll: { flex: 1 },
  content: {
    padding: 16,
    gap: 16,
  },
  uploadArea: {
    height: 140,
    borderRadius: 14,
    borderWidth: 1.5,
    borderStyle: "dashed",
    alignItems: "center",
    justifyContent: "center",
    gap: 6,
  },
  uploadText: {
    fontSize: 15,
    fontFamily: "Inter_600SemiBold",
  },
  uploadHint: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
  },
  formSection: { gap: 6 },
  formLabel: {
    fontSize: 12,
    fontFamily: "Inter_600SemiBold",
    textTransform: "uppercase",
    letterSpacing: 0.5,
  },
  textInput: {
    borderWidth: 1,
    borderRadius: 12,
    paddingHorizontal: 14,
    paddingVertical: 12,
    fontSize: 15,
    fontFamily: "Inter_400Regular",
  },
  textArea: {
    height: 100,
    textAlignVertical: "top",
  },
  chipRow: { gap: 8, paddingRight: 8 },
  chip: {
    flexDirection: "row",
    alignItems: "center",
    gap: 5,
    paddingHorizontal: 12,
    paddingVertical: 8,
    borderRadius: 20,
  },
  chipLabel: { fontSize: 13, fontFamily: "Inter_500Medium" },
  conditionRow: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  conditionChip: {
    paddingHorizontal: 14,
    paddingVertical: 8,
    borderRadius: 20,
    borderWidth: 1,
  },
  conditionLabel: { fontSize: 13, fontFamily: "Inter_500Medium" },
  auctionToggle: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    padding: 14,
    borderRadius: 12,
    borderWidth: 1,
  },
  toggleLabel: { fontSize: 15, fontFamily: "Inter_600SemiBold" },
  toggleHint: { fontSize: 12, fontFamily: "Inter_400Regular", marginTop: 2 },
  warningBanner: {
    borderWidth: 1,
    borderRadius: 10,
    padding: 12,
  },
  warningText: {
    fontSize: 13,
    fontFamily: "Inter_400Regular",
    color: "#92400E",
    lineHeight: 20,
  },
  priceRow: { flexDirection: "row", gap: 8, alignItems: "center" },
  currencyBtn: {
    flexDirection: "row",
    alignItems: "center",
    gap: 4,
    paddingHorizontal: 12,
    paddingVertical: 12,
    borderRadius: 12,
    borderWidth: 1,
  },
  currencyCode: { fontSize: 15, fontFamily: "Inter_600SemiBold" },
  priceInput: { flex: 1 },
  locationRow: {
    flexDirection: "row",
    alignItems: "center",
  },
  locationInput: { flex: 1 },
  geoIndicator: { marginLeft: 8 },
  geoHint: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
    marginTop: 4,
  },
  sectionTitle: {
    fontSize: 11,
    fontFamily: "Inter_600SemiBold",
    letterSpacing: 0.8,
    marginTop: 4,
  },
  fieldError: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
    color: "#EF4444",
  },
  submitBtn: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: 8,
    paddingVertical: 16,
    borderRadius: 14,
    marginTop: 4,
  },
  submitText: {
    fontSize: 16,
    fontFamily: "Inter_600SemiBold",
    color: "#fff",
  },
  modalContainer: { flex: 1 },
  modalHeader: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    padding: 20,
    borderBottomWidth: 1,
  },
  modalTitle: {
    fontSize: 18,
    fontFamily: "Inter_700Bold",
  },
  currencyOption: {
    flexDirection: "row",
    alignItems: "center",
    gap: 14,
    paddingHorizontal: 20,
    paddingVertical: 16,
    borderBottomWidth: 1,
  },
  currencySymbol: {
    fontSize: 18,
    fontFamily: "Inter_700Bold",
    width: 30,
    textAlign: "center",
  },
  currencyName: {
    fontSize: 15,
    fontFamily: "Inter_500Medium",
  },
  currencyCodeLabel: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
  },
});
