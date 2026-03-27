import { Feather } from "@expo/vector-icons";
import * as Haptics from "expo-haptics";
import { router, useLocalSearchParams } from "expo-router";
import React, { useEffect, useRef, useState } from "react";
import {
  FlatList,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
  useColorScheme,
} from "react-native";
import { KeyboardAvoidingView } from "react-native-keyboard-controller";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import Colors from "@/constants/colors";
import { useAppContext, type Message } from "@/context/AppContext";
import { formatRelativeTime } from "@/utils/format";

export default function ConversationScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const colorScheme = useColorScheme();
  const colors = Colors[colorScheme ?? "light"];
  const insets = useSafeAreaInsets();
  const isWeb = Platform.OS === "web";
  const { conversations, sendMessage, markConversationRead, user } =
    useAppContext();

  const conversation = conversations.find((c) => c.id === id);
  const [text, setText] = useState("");
  const flatListRef = useRef<FlatList>(null);

  useEffect(() => {
    if (id) {
      markConversationRead(id);
    }
  }, [id]);

  if (!conversation) {
    return (
      <View style={[styles.notFound, { backgroundColor: colors.background }]}>
        <Text style={[{ color: colors.text }]}>Conversation not found</Text>
        <Pressable onPress={() => router.back()}>
          <Text style={[{ color: colors.tint }]}>Go back</Text>
        </Pressable>
      </View>
    );
  }

  const handleSend = () => {
    const trimmed = text.trim();
    if (!trimmed) return;
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    sendMessage(conversation.id, trimmed);
    setText("");
  };

  const renderMessage = ({ item, index }: { item: Message; index: number }) => {
    const isMe = item.senderId === user.id;
    const prevMsg = conversation.messages[index - 1];
    const showTime =
      !prevMsg ||
      new Date(item.createdAt).getTime() -
        new Date(prevMsg.createdAt).getTime() >
        300000;

    return (
      <View>
        {showTime && (
          <Text style={[styles.timestamp, { color: colors.textTertiary }]}>
            {formatRelativeTime(item.createdAt)}
          </Text>
        )}
        <View
          style={[
            styles.messageRow,
            isMe ? styles.messageRowMe : styles.messageRowOther,
          ]}
        >
          {!isMe && (
            <View style={[styles.msgAvatar, { backgroundColor: colors.tint }]}>
              <Feather name="user" size={12} color="#fff" />
            </View>
          )}
          <View
            style={[
              styles.bubble,
              isMe
                ? [styles.bubbleMe, { backgroundColor: colors.tint }]
                : [styles.bubbleOther, { backgroundColor: colors.backgroundSecondary, borderColor: colors.border }],
            ]}
          >
            <Text style={[styles.bubbleText, { color: isMe ? "#fff" : colors.text }]}>
              {item.text}
            </Text>
          </View>
        </View>
      </View>
    );
  };

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <View
        style={[
          styles.header,
          {
            paddingTop: isWeb ? 67 : insets.top + 8,
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
        <View style={styles.headerInfo}>
          <View style={[styles.headerAvatar, { backgroundColor: colors.tint }]}>
            <Feather name="user" size={14} color="#fff" />
          </View>
          <View>
            <Text style={[styles.headerName, { color: colors.text }]}>
              {conversation.otherUserName}
            </Text>
            <Text
              style={[styles.headerListing, { color: colors.textTertiary }]}
              numberOfLines={1}
            >
              Re: {conversation.listingTitle}
            </Text>
          </View>
        </View>
        <Pressable style={({ pressed }) => [styles.moreBtn, pressed && { opacity: 0.7 }]}>
          <Feather name="more-vertical" size={20} color={colors.text} />
        </Pressable>
      </View>

      <KeyboardAvoidingView
        style={styles.flex}
        behavior="padding"
        keyboardVerticalOffset={0}
      >
        <FlatList
          ref={flatListRef}
          data={conversation.messages}
          keyExtractor={(item) => item.id}
          renderItem={renderMessage}
          contentContainerStyle={styles.messageList}
          showsVerticalScrollIndicator={false}
          onContentSizeChange={() =>
            flatListRef.current?.scrollToEnd({ animated: true })
          }
          ListEmptyComponent={
            <View style={styles.emptyConv}>
              <Feather name="message-circle" size={32} color={colors.textTertiary} />
              <Text style={[styles.emptyConvText, { color: colors.textTertiary }]}>
                Start the conversation
              </Text>
            </View>
          }
        />

        <View
          style={[
            styles.inputBar,
            {
              paddingBottom: isWeb ? 34 : insets.bottom + 8,
              backgroundColor: colors.backgroundSecondary,
              borderTopColor: colors.border,
            },
          ]}
        >
          <View
            style={[
              styles.inputContainer,
              { backgroundColor: colors.backgroundTertiary },
            ]}
          >
            <TextInput
              style={[styles.input, { color: colors.text }]}
              placeholder="Type a message..."
              placeholderTextColor={colors.textTertiary}
              value={text}
              onChangeText={setText}
              multiline
              maxLength={500}
            />
          </View>
          <Pressable
            onPress={handleSend}
            disabled={!text.trim()}
            style={({ pressed }) => [
              styles.sendBtn,
              {
                backgroundColor: text.trim() ? colors.tint : colors.backgroundTertiary,
                opacity: pressed ? 0.85 : 1,
              },
            ]}
          >
            <Feather
              name="send"
              size={18}
              color={text.trim() ? "#fff" : colors.textTertiary}
            />
          </Pressable>
        </View>
      </KeyboardAvoidingView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  flex: { flex: 1 },
  notFound: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: 12,
  },
  header: {
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: 12,
    paddingBottom: 12,
    borderBottomWidth: 1,
    gap: 10,
  },
  backBtn: {
    width: 36,
    height: 36,
    alignItems: "center",
    justifyContent: "center",
  },
  headerInfo: {
    flex: 1,
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
  },
  headerAvatar: {
    width: 36,
    height: 36,
    borderRadius: 18,
    alignItems: "center",
    justifyContent: "center",
  },
  headerName: {
    fontSize: 15,
    fontFamily: "Inter_600SemiBold",
  },
  headerListing: {
    fontSize: 12,
    fontFamily: "Inter_400Regular",
    maxWidth: 180,
  },
  moreBtn: {
    width: 36,
    height: 36,
    alignItems: "center",
    justifyContent: "center",
  },
  messageList: {
    padding: 16,
    gap: 4,
    flexGrow: 1,
  },
  timestamp: {
    textAlign: "center",
    fontSize: 11,
    fontFamily: "Inter_400Regular",
    marginVertical: 10,
  },
  messageRow: {
    flexDirection: "row",
    alignItems: "flex-end",
    marginBottom: 4,
    gap: 6,
  },
  messageRowMe: {
    justifyContent: "flex-end",
  },
  messageRowOther: {
    justifyContent: "flex-start",
  },
  msgAvatar: {
    width: 24,
    height: 24,
    borderRadius: 12,
    alignItems: "center",
    justifyContent: "center",
  },
  bubble: {
    maxWidth: "75%",
    paddingHorizontal: 14,
    paddingVertical: 10,
    borderRadius: 18,
  },
  bubbleMe: {
    borderBottomRightRadius: 4,
  },
  bubbleOther: {
    borderBottomLeftRadius: 4,
    borderWidth: 1,
  },
  bubbleText: {
    fontSize: 15,
    fontFamily: "Inter_400Regular",
    lineHeight: 22,
  },
  emptyConv: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: 8,
    paddingTop: 60,
  },
  emptyConvText: {
    fontSize: 14,
    fontFamily: "Inter_400Regular",
  },
  inputBar: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: 10,
    paddingHorizontal: 12,
    paddingTop: 8,
    borderTopWidth: 1,
  },
  inputContainer: {
    flex: 1,
    borderRadius: 22,
    paddingHorizontal: 14,
    paddingVertical: 10,
    maxHeight: 120,
  },
  input: {
    fontSize: 15,
    fontFamily: "Inter_400Regular",
  },
  sendBtn: {
    width: 42,
    height: 42,
    borderRadius: 21,
    alignItems: "center",
    justifyContent: "center",
  },
});
