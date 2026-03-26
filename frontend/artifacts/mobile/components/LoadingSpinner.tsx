import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";

interface LoadingSpinnerProps {
  size?: "small" | "large";
  color?: string;
}

export default function LoadingSpinner({
  size = "large",
  color = "#0071CE",
}: LoadingSpinnerProps) {
  return (
    <View style={styles.container}>
      <ActivityIndicator size={size} color={color} />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
  },
});
