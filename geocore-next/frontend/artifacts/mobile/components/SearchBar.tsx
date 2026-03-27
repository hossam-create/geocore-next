import { Feather } from "@expo/vector-icons";
import React from "react";
import {
  Pressable,
  StyleSheet,
  Text,
  View,
  useColorScheme,
} from "react-native";

import Colors from "@/constants/colors";

interface SearchBarProps {
  onPress: () => void;
  placeholder?: string;
}

export default function SearchBar({
  onPress,
  placeholder = "Search for anything...",
}: SearchBarProps) {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];

  return (
    <Pressable
      onPress={onPress}
      style={({ pressed }) => [
        styles.container,
        {
          backgroundColor: colors.backgroundSecondary,
          borderColor: colors.border,
          opacity: pressed ? 0.9 : 1,
        },
      ]}
    >
      <Feather name="search" size={18} color={colors.textTertiary} />
      <Text style={[styles.placeholder, { color: colors.textTertiary }]}>
        {placeholder}
      </Text>
      <View style={[styles.searchBtn, { backgroundColor: "#FFC220" }]}>
        <Text style={styles.searchBtnText}>Search</Text>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  container: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    paddingHorizontal: 14,
    paddingVertical: 11,
    borderRadius: 12,
    borderWidth: 1,
  },
  placeholder: {
    flex: 1,
    fontSize: 15,
    fontFamily: "Inter_400Regular",
  },
  searchBtn: {
    borderRadius: 6,
    paddingHorizontal: 10,
    paddingVertical: 4,
  },
  searchBtnText: {
    fontSize: 12,
    fontFamily: "Inter_700Bold",
    color: "#1A1A1A",
  },
});
