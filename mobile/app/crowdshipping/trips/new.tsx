import { router } from "expo-router";
import React, { useState } from "react";
import {
  ActivityIndicator,
  Alert,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
  useColorScheme,
} from "react-native";

import ScreenHeader from "@/components/ScreenHeader";
import Colors from "@/constants/colors";
import { crowdshippingAPI } from "@/utils/api";

export default function NewTripScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];

  const [originCountry, setOriginCountry] = useState("");
  const [originCity, setOriginCity] = useState("");
  const [destCountry, setDestCountry] = useState("");
  const [destCity, setDestCity] = useState("");
  const [departureDate, setDepartureDate] = useState("");
  const [arrivalDate, setArrivalDate] = useState("");
  const [availableWeight, setAvailableWeight] = useState("");
  const [maxItems, setMaxItems] = useState("5");
  const [pricePerKg, setPricePerKg] = useState("");
  const [basePrice, setBasePrice] = useState("");
  const [currency, setCurrency] = useState("AED");
  const [notes, setNotes] = useState("");

  const [submitting, setSubmitting] = useState(false);

  const submit = async () => {
    if (!originCountry || !originCity || !destCountry || !destCity) {
      Alert.alert("Missing info", "Origin and destination are required.");
      return;
    }
    if (!departureDate || !arrivalDate) {
      Alert.alert("Missing info", "Departure and arrival dates are required (YYYY-MM-DD).");
      return;
    }
    const dep = toRFC3339(departureDate);
    const arr = toRFC3339(arrivalDate);
    if (!dep || !arr) {
      Alert.alert("Invalid date", "Use YYYY-MM-DD format.");
      return;
    }
    setSubmitting(true);
    try {
      await crowdshippingAPI.createTrip({
        origin_country: originCountry.trim(),
        origin_city: originCity.trim(),
        dest_country: destCountry.trim(),
        dest_city: destCity.trim(),
        departure_date: dep,
        arrival_date: arr,
        available_weight: parseFloat(availableWeight) || 0,
        max_items: parseInt(maxItems, 10) || 5,
        price_per_kg: parseFloat(pricePerKg) || 0,
        base_price: parseFloat(basePrice) || 0,
        currency: currency.trim() || "AED",
        notes: notes.trim(),
        frequency: "one-time",
      });
      Alert.alert("Trip posted", "Your trip is now visible to shoppers.", [
        { text: "OK", onPress: () => router.replace("/crowdshipping/trips/my") },
      ]);
    } catch (err: any) {
      const msg = err?.response?.data?.error ?? err?.message ?? "Failed to create trip";
      Alert.alert("Error", msg);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <ScreenHeader title="Post a Trip" />
      <KeyboardAvoidingView
        style={{ flex: 1 }}
        behavior={Platform.OS === "ios" ? "padding" : undefined}
      >
        <ScrollView contentContainerStyle={styles.form} showsVerticalScrollIndicator={false}>
          <SectionTitle text="From" colors={colors} />
          <LabeledInput
            label="Origin country"
            value={originCountry}
            onChangeText={setOriginCountry}
            placeholder="e.g. AE"
            colors={colors}
          />
          <LabeledInput
            label="Origin city"
            value={originCity}
            onChangeText={setOriginCity}
            placeholder="e.g. Dubai"
            colors={colors}
          />

          <SectionTitle text="To" colors={colors} />
          <LabeledInput
            label="Destination country"
            value={destCountry}
            onChangeText={setDestCountry}
            placeholder="e.g. EG"
            colors={colors}
          />
          <LabeledInput
            label="Destination city"
            value={destCity}
            onChangeText={setDestCity}
            placeholder="e.g. Cairo"
            colors={colors}
          />

          <SectionTitle text="Travel dates" colors={colors} />
          <LabeledInput
            label="Departure (YYYY-MM-DD)"
            value={departureDate}
            onChangeText={setDepartureDate}
            placeholder="2026-05-01"
            colors={colors}
          />
          <LabeledInput
            label="Arrival (YYYY-MM-DD)"
            value={arrivalDate}
            onChangeText={setArrivalDate}
            placeholder="2026-05-03"
            colors={colors}
          />

          <SectionTitle text="Capacity & pricing" colors={colors} />
          <LabeledInput
            label="Available weight (kg)"
            value={availableWeight}
            onChangeText={setAvailableWeight}
            keyboardType="decimal-pad"
            placeholder="10"
            colors={colors}
          />
          <LabeledInput
            label="Max items"
            value={maxItems}
            onChangeText={setMaxItems}
            keyboardType="number-pad"
            placeholder="5"
            colors={colors}
          />
          <LabeledInput
            label="Price per kg"
            value={pricePerKg}
            onChangeText={setPricePerKg}
            keyboardType="decimal-pad"
            placeholder="50"
            colors={colors}
          />
          <LabeledInput
            label="Base price"
            value={basePrice}
            onChangeText={setBasePrice}
            keyboardType="decimal-pad"
            placeholder="0"
            colors={colors}
          />
          <LabeledInput
            label="Currency"
            value={currency}
            onChangeText={setCurrency}
            placeholder="AED"
            colors={colors}
          />

          <SectionTitle text="Notes" colors={colors} />
          <LabeledInput
            label="Anything shoppers should know?"
            value={notes}
            onChangeText={setNotes}
            placeholder="e.g. I can accept fragile items"
            multiline
            colors={colors}
          />

          <Pressable
            onPress={submit}
            disabled={submitting}
            style={({ pressed }) => [
              styles.submitBtn,
              { backgroundColor: colors.tint, opacity: pressed || submitting ? 0.8 : 1 },
            ]}
          >
            {submitting ? (
              <ActivityIndicator color="#fff" />
            ) : (
              <Text style={styles.submitText}>Post trip</Text>
            )}
          </Pressable>
        </ScrollView>
      </KeyboardAvoidingView>
    </View>
  );
}

function SectionTitle({ text, colors }: { text: string; colors: (typeof Colors)["light"] }) {
  return (
    <Text style={[styles.sectionTitle, { color: colors.textSecondary }]}>
      {text.toUpperCase()}
    </Text>
  );
}

function LabeledInput({
  label,
  value,
  onChangeText,
  placeholder,
  keyboardType,
  multiline,
  colors,
}: {
  label: string;
  value: string;
  onChangeText: (v: string) => void;
  placeholder?: string;
  keyboardType?: "default" | "decimal-pad" | "number-pad" | "email-address";
  multiline?: boolean;
  colors: (typeof Colors)["light"];
}) {
  return (
    <View style={styles.field}>
      <Text style={[styles.label, { color: colors.textSecondary }]}>{label}</Text>
      <TextInput
        value={value}
        onChangeText={onChangeText}
        placeholder={placeholder}
        placeholderTextColor={colors.textTertiary}
        keyboardType={keyboardType ?? "default"}
        multiline={multiline}
        style={[
          styles.input,
          multiline && { minHeight: 80, textAlignVertical: "top" },
          {
            backgroundColor: colors.backgroundSecondary,
            borderColor: colors.border,
            color: colors.text,
          },
        ]}
      />
    </View>
  );
}

function toRFC3339(yyyymmdd: string): string | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(yyyymmdd.trim());
  if (!match) return null;
  const d = new Date(`${yyyymmdd.trim()}T00:00:00Z`);
  if (Number.isNaN(d.getTime())) return null;
  return d.toISOString();
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  form: { padding: 16, gap: 10 },
  sectionTitle: {
    fontSize: 11,
    fontFamily: "Inter_600SemiBold",
    letterSpacing: 0.6,
    marginTop: 14,
    marginBottom: 4,
  },
  field: { gap: 6 },
  label: { fontSize: 12, fontFamily: "Inter_500Medium" },
  input: {
    borderWidth: 1,
    borderRadius: 10,
    paddingHorizontal: 12,
    paddingVertical: 10,
    fontSize: 15,
    fontFamily: "Inter_400Regular",
  },
  submitBtn: {
    marginTop: 24,
    paddingVertical: 14,
    borderRadius: 12,
    alignItems: "center",
  },
  submitText: {
    color: "#fff",
    fontSize: 16,
    fontFamily: "Inter_600SemiBold",
  },
});
