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

export default function NewDeliveryRequestScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];

  const [itemName, setItemName] = useState("");
  const [itemDescription, setItemDescription] = useState("");
  const [itemURL, setItemURL] = useState("");
  const [itemPrice, setItemPrice] = useState("");
  const [itemWeight, setItemWeight] = useState("");
  const [pickupCountry, setPickupCountry] = useState("");
  const [pickupCity, setPickupCity] = useState("");
  const [deliveryCountry, setDeliveryCountry] = useState("");
  const [deliveryCity, setDeliveryCity] = useState("");
  const [reward, setReward] = useState("");
  const [currency, setCurrency] = useState("AED");
  const [notes, setNotes] = useState("");

  const [submitting, setSubmitting] = useState(false);

  const submit = async () => {
    if (!itemName.trim()) {
      Alert.alert("Missing info", "Item name is required.");
      return;
    }
    const price = parseFloat(itemPrice);
    if (!price || price <= 0) {
      Alert.alert("Missing info", "Item price must be greater than 0.");
      return;
    }
    const rewardAmount = parseFloat(reward);
    if (!rewardAmount || rewardAmount <= 0) {
      Alert.alert("Missing info", "Reward must be greater than 0.");
      return;
    }
    if (!pickupCountry.trim() || !pickupCity.trim() || !deliveryCountry.trim() || !deliveryCity.trim()) {
      Alert.alert("Missing info", "Pickup and delivery locations are required.");
      return;
    }
    setSubmitting(true);
    try {
      const payload: Record<string, unknown> = {
        item_name: itemName.trim(),
        item_description: itemDescription.trim(),
        item_url: itemURL.trim(),
        item_price: price,
        pickup_country: pickupCountry.trim(),
        pickup_city: pickupCity.trim(),
        delivery_country: deliveryCountry.trim(),
        delivery_city: deliveryCity.trim(),
        reward: rewardAmount,
        currency: currency.trim() || "AED",
        notes: notes.trim(),
      };
      const weight = parseFloat(itemWeight);
      if (weight > 0) payload.item_weight = weight;

      await crowdshippingAPI.createDeliveryRequest(payload);
      Alert.alert("Request posted", "Travelers can now see your request.", [
        { text: "OK", onPress: () => router.replace("/crowdshipping/requests/my") },
      ]);
    } catch (err: any) {
      const msg = err?.response?.data?.error ?? err?.message ?? "Failed to post request";
      Alert.alert("Error", msg);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <ScreenHeader title="Request a Delivery" />
      <KeyboardAvoidingView
        style={{ flex: 1 }}
        behavior={Platform.OS === "ios" ? "padding" : undefined}
      >
        <ScrollView contentContainerStyle={styles.form} showsVerticalScrollIndicator={false}>
          <SectionTitle text="Item" colors={colors} />
          <LabeledInput label="Item name" value={itemName} onChangeText={setItemName} placeholder="e.g. iPhone 16 Pro" colors={colors} />
          <LabeledInput
            label="Description (optional)"
            value={itemDescription}
            onChangeText={setItemDescription}
            placeholder="Details, size, color..."
            multiline
            colors={colors}
          />
          <LabeledInput
            label="Item URL (optional)"
            value={itemURL}
            onChangeText={setItemURL}
            placeholder="https://..."
            colors={colors}
          />
          <LabeledInput label="Item price" value={itemPrice} onChangeText={setItemPrice} keyboardType="decimal-pad" placeholder="1500" colors={colors} />
          <LabeledInput label="Weight (kg, optional)" value={itemWeight} onChangeText={setItemWeight} keyboardType="decimal-pad" placeholder="0.5" colors={colors} />

          <SectionTitle text="Pickup" colors={colors} />
          <LabeledInput label="Pickup country" value={pickupCountry} onChangeText={setPickupCountry} placeholder="e.g. US" colors={colors} />
          <LabeledInput label="Pickup city" value={pickupCity} onChangeText={setPickupCity} placeholder="e.g. New York" colors={colors} />

          <SectionTitle text="Delivery" colors={colors} />
          <LabeledInput label="Delivery country" value={deliveryCountry} onChangeText={setDeliveryCountry} placeholder="e.g. AE" colors={colors} />
          <LabeledInput label="Delivery city" value={deliveryCity} onChangeText={setDeliveryCity} placeholder="e.g. Dubai" colors={colors} />

          <SectionTitle text="Reward" colors={colors} />
          <LabeledInput label="Reward amount" value={reward} onChangeText={setReward} keyboardType="decimal-pad" placeholder="100" colors={colors} />
          <LabeledInput label="Currency" value={currency} onChangeText={setCurrency} placeholder="AED" colors={colors} />

          <SectionTitle text="Notes" colors={colors} />
          <LabeledInput
            label="Anything travelers should know?"
            value={notes}
            onChangeText={setNotes}
            placeholder="Preferred delivery date, fragile, etc."
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
              <Text style={styles.submitText}>Post request</Text>
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
