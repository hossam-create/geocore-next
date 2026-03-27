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
import { useAppContext, type Conversation } from "@/context/AppContext";
import { formatRelativeTime } from "@/utils/format";

export default function MessagesScreen() {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const insets = useSafeAreaInsets();
  const isWeb = Platform.OS === "web";
  const topPad = isWeb ? 67 : insets.top;
  const { conversations } = useAppContext();

  const sortedConvs = [...conversations].sort(
    (a, b) =>
      new Date(b.lastMessageAt).getTime() - new Date(a.lastMessageAt).getTime()
  );

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
        <Text style={[styles.headerTitle, { color: colors.text }]}>Messages</Text>
      </View>

      <FlatList
        data={sortedConvs}
        keyExtractor={(item) => item.id}
        contentInsetAdjustmentBehavior="automatic"
        showsVerticalScrollIndicator={false}
        renderItem={({ item }) => (
          <ConversationRow conversation={item} />
        )}
        ItemSeparatorComponent={() => (
          <View style={[styles.separator, { backgroundColor: colors.border }]} />
        )}
        ListEmptyComponent={
          <View style={styles.empty}>
            <Feather name="message-circle" size={40} color={colors.textTertiary} />
            <Text style={[styles.emptyTitle, { color: colors.text }]}>
              No conversations yet
            </Text>
            <Text style={[styles.emptyText, { color: colors.textTertiary }]}>
              Message sellers when you find something you like
            </Text>
          </View>
        }
        ListFooterComponent={<View style={{ height: isWeb ? 34 : 100 }} />}
      />
    </View>
  );
}

function ConversationRow({ conversation }: { conversation: Conversation }) {
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const hasUnread = conversation.unreadCount > 0;

  return (
    <Pressable
      onPress={() =>
        router.push({
          pathname: "/conversation/[id]",
          params: { id: conversation.id },
        })
      }
      style={({ pressed }) => [
        styles.row,
        {
          backgroundColor: hasUnread
            ? colorScheme === "dark"
              ? colors.backgroundSecondary
              : "#FFF8F6"
            : colors.backgroundSecondary,
          opacity: pressed ? 0.85 : 1,
        },
      ]}
    >
      <View style={[styles.avatar, { backgroundColor: colors.tint }]}>
        <Feather name="user" size={20} color="#fff" />
      </View>

      <View style={styles.rowContent}>
        <View style={styles.rowTop}>
          <Text
            style={[
              styles.name,
              { color: colors.text, fontFamily: hasUnread ? "Inter_700Bold" : "Inter_600SemiBold" },
            ]}
          >
            {conversation.otherUserName}
          </Text>
          <Text style={[styles.time, { color: colors.textTertiary }]}>
            {formatRelativeTime(conversation.lastMessageAt)}
          </Text>
        </View>

        <Text
          style={[styles.listingTitle, { color: colors.textTertiary }]}
          numberOfLines={1}
        >
          Re: {conversation.listingTitle}
        </Text>

        <View style={styles.rowBottom}>
          <Text
            style={[
              styles.lastMessage,
              {
                color: hasUnread ? colors.text : colors.textSecondary,
                fontFamily: hasUnread ? "Inter_500Medium" : "Inter_400Regular",
                flex: 1,
              },
            ]}
            numberOfLines={1}
          >
            {conversation.lastMessage}
          </Text>
          {hasUnread && (
            <View style={[styles.badge, { backgroundColor: colors.tint }]}>
              <Text style={styles.badgeText}>{conversation.unreadCount}</Text>
            </View>
          )}
        </View>
      </View>
    </Pressable>
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
  row: {
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: 16,
    paddingVertical: 14,
    gap: 12,
  },
  avatar: {
    width: 48,
    height: 48,
    borderRadius: 24,
    alignItems: "center",
    justifyContent: "center",
    flexShrink: 0,
  },
  rowContent: {
    flex: 1,
    gap: 3,
  },
  rowTop: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
  },
  name: {
    fontSize: 15,
  },
  time: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
  },
  listingTitle: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
  },
  rowBottom: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
  },
  lastMessage: {
    fontSize: 14,
  },
  badge: {
    minWidth: 20,
    height: 20,
    borderRadius: 10,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 5,
  },
  badgeText: {
    fontSize: 11,
    fontFamily: "Inter_700Bold",
    color: "#fff",
  },
  separator: {
    height: 1,
    marginLeft: 76,
  },
  empty: {
    alignItems: "center",
    justifyContent: "center",
    paddingTop: 100,
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
