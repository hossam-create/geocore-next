'use client'
import { useState, useEffect, useRef, useCallback } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { X, Send, MessageCircle, Loader2 } from "lucide-react";
import { useAuthStore } from "@/store/auth";

interface Message {
  id: string;
  conversation_id: string;
  sender_id: string;
  content: string;
  type: string;
  created_at: string;
}

interface Conversation {
  id: string;
  listing_id?: string;
  members: { user_id: string }[];
}

interface ChatPanelProps {
  sellerId: string;
  sellerName: string;
  listingId?: string;
  onClose: () => void;
}

function buildWsUrl(conversationId: string): string {
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  const token = localStorage.getItem("access_token") ?? "";
  return `${proto}//${window.location.host}/api/v1/chat/conversations/${conversationId}/ws?token=${encodeURIComponent(token)}`;
}

export function ChatPanel({ sellerId, sellerName, listingId, onClose }: ChatPanelProps) {
  const { user } = useAuthStore();
  const qc = useQueryClient();
  const [input, setInput] = useState("");
  const wsRef = useRef<WebSocket | null>(null);
  const bottomRef = useRef<HTMLDivElement | null>(null);
  const [convId, setConvId] = useState<string | null>(null);
  const [wsConnected, setWsConnected] = useState(false);

  const { mutate: initConv, isPending: convLoading } = useMutation({
    mutationFn: () =>
      api.post("/chat/conversations", {
        other_user_id: sellerId,
        listing_id: listingId ?? null,
      }).then((r) => r.data.data ?? r.data),
    onSuccess: (conv: Conversation) => {
      setConvId(conv.id);
    },
  });

  const { data: messages = [], isLoading: msgsLoading } = useQuery<Message[]>({
    queryKey: ["chat-messages", convId],
    queryFn: () =>
      api.get(`/chat/conversations/${convId}/messages`).then((r) => r.data.data ?? r.data ?? []),
    enabled: !!convId,
    staleTime: 0,
  });

  const { mutate: sendMsg, isPending: sending } = useMutation({
    mutationFn: (content: string) =>
      api.post(`/chat/conversations/${convId}/messages`, { content, type: "text" }).then((r) => r.data.data ?? r.data),
    onSuccess: (msg: Message) => {
      qc.setQueryData<Message[]>(["chat-messages", convId], (old) => {
        if (!old) return [msg];
        if (old.find((m) => m.id === msg.id)) return old;
        return [...old, msg];
      });
    },
  });

  useEffect(() => {
    initConv();
  }, []);

  useEffect(() => {
    if (!convId) return;

    const ws = new WebSocket(buildWsUrl(convId));
    wsRef.current = ws;

    ws.onopen = () => {
      setWsConnected(true);
    };
    ws.onclose = () => setWsConnected(false);
    ws.onerror = () => setWsConnected(false);

    ws.onmessage = (e) => {
      try {
        const msg: Message = JSON.parse(e.data);
        if (!msg.id || !msg.content) return;
        if (msg.sender_id === user?.id) return;
        qc.setQueryData<Message[]>(["chat-messages", convId], (old) => {
          if (!old) return [msg];
          if (old.find((m) => m.id === msg.id)) return old;
          return [...old, msg];
        });
      } catch {
      }
    };

    return () => {
      ws.close();
      wsRef.current = null;
    };
  }, [convId, user?.id]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const handleSend = useCallback(() => {
    const text = input.trim();
    if (!text || !convId) return;
    setInput("");
    sendMsg(text);
  }, [input, convId, sendMsg]);

  const handleKey = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="fixed bottom-6 right-6 w-[360px] max-h-[520px] bg-white rounded-2xl shadow-2xl border border-gray-200 flex flex-col z-50 overflow-hidden">
      <div className="flex items-center gap-3 px-4 py-3 bg-[#0071CE] text-white">
        <div className="w-8 h-8 rounded-full bg-white/20 flex items-center justify-center font-bold text-sm shrink-0">
          {sellerName[0]?.toUpperCase()}
        </div>
        <div className="flex-1 min-w-0">
          <p className="font-semibold text-sm truncate">{sellerName}</p>
          <p className="text-xs text-white/70">{wsConnected ? "Connected" : "Connecting..."}</p>
        </div>
        <button onClick={onClose} className="p-1 hover:bg-white/20 rounded-lg transition-colors">
          <X size={16} />
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-3 bg-gray-50 min-h-[300px] max-h-[380px]">
        {(convLoading || msgsLoading) ? (
          <div className="flex items-center justify-center h-full">
            <Loader2 size={24} className="animate-spin text-gray-300" />
          </div>
        ) : messages.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-gray-400 gap-2">
            <MessageCircle size={32} className="opacity-40" />
            <p className="text-sm">Start a conversation with {sellerName}</p>
          </div>
        ) : (
          messages.map((msg) => {
            const isMine = msg.sender_id === user?.id;
            return (
              <div key={msg.id} className={`flex ${isMine ? "justify-end" : "justify-start"}`}>
                <div
                  className={`max-w-[75%] px-3 py-2 rounded-2xl text-sm leading-relaxed ${
                    isMine
                      ? "bg-[#0071CE] text-white rounded-br-sm"
                      : "bg-white text-gray-800 border border-gray-200 rounded-bl-sm"
                  }`}
                >
                  {msg.content}
                  <p className={`text-[10px] mt-1 ${isMine ? "text-white/60" : "text-gray-400"}`}>
                    {new Date(msg.created_at).toLocaleTimeString("en-AE", { hour: "2-digit", minute: "2-digit" })}
                  </p>
                </div>
              </div>
            );
          })
        )}
        <div ref={bottomRef} />
      </div>

      <div className="px-3 py-3 border-t border-gray-100 flex items-end gap-2 bg-white">
        <textarea
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKey}
          placeholder="Type a message..."
          rows={1}
          className="flex-1 resize-none text-sm border border-gray-200 rounded-xl px-3 py-2 focus:outline-none focus:border-[#0071CE] max-h-[80px] bg-gray-50"
          disabled={!convId || sending}
        />
        <button
          onClick={handleSend}
          disabled={!input.trim() || !convId || sending}
          className="w-9 h-9 bg-[#0071CE] text-white rounded-xl flex items-center justify-center hover:bg-[#005BA1] transition-colors disabled:opacity-40 shrink-0"
        >
          {sending ? <Loader2 size={14} className="animate-spin" /> : <Send size={14} />}
        </button>
      </div>
    </div>
  );
}
