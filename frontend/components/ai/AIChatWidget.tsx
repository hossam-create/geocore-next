'use client';

import { useState, useEffect, useRef } from 'react';
import Link from 'next/link';
import { Send, X, Minimize2, Maximize2, Sparkles } from 'lucide-react';
interface Message {
  role: 'user' | 'assistant';
  content: string;
  actions?: SuggestedAction[];
}

interface SuggestedAction {
  type: string;
  label: string;
  url?: string;
}

async function sendChatMessage(message: string, history: Message[], token: string | null) {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (token) headers['Authorization'] = `Bearer ${token}`;

  const res = await fetch('/api/v1/ai/chat', {
    method: 'POST',
    headers,
    body: JSON.stringify({
      message,
      history: history.map((m) => ({ role: m.role, content: m.content })),
    }),
  });

  if (!res.ok) throw new Error('Chat request failed');
  const json = await res.json();
  return json.data ?? json;
}

export default function AIChatWidget() {
  const [isOpen, setIsOpen] = useState(false);
  const [isMinimized, setIsMinimized] = useState(false);
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [isTyping, setIsTyping] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const token = typeof window !== 'undefined' ? localStorage.getItem('access_token') : null;

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, isTyping]);

  useEffect(() => {
    if (isOpen && messages.length === 0) {
      setMessages([
        {
          role: 'assistant',
          content: "👋 Hi! I'm your Mnbarh AI assistant.\nHow can I help you today?",
          actions: [
            { type: 'navigate', label: 'Browse Listings', url: '/listings' },
            { type: 'navigate', label: 'Help Center', url: '/help' },
          ],
        },
      ]);
    }
  }, [isOpen]);

  const handleSend = async () => {
    const trimmed = input.trim();
    if (!trimmed || isTyping) return;

    const userMsg: Message = { role: 'user', content: trimmed };
    setMessages((prev) => [...prev, userMsg]);
    setInput('');
    setIsTyping(true);

    try {
      const resp = await sendChatMessage(trimmed, messages, token ?? null);
      setMessages((prev) => [
        ...prev,
        {
          role: 'assistant',
          content: resp.message ?? "Sorry, I couldn't process that.",
          actions: resp.actions ?? [],
        },
      ]);
    } catch {
      setMessages((prev) => [
        ...prev,
        { role: 'assistant', content: '😔 Something went wrong. Please try again.' },
      ]);
    } finally {
      setIsTyping(false);
    }
  };

  const handleKey = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  // ── Floating button (closed state) ──────────────────────────────────────────
  if (!isOpen) {
    return (
      <button
        onClick={() => setIsOpen(true)}
        className="fixed bottom-6 right-6 w-14 h-14 bg-gradient-to-br from-violet-600 to-blue-600 rounded-full shadow-lg hover:shadow-xl transition-all duration-200 flex items-center justify-center z-50 group"
        aria-label="Open AI assistant"
      >
        <Sparkles className="w-6 h-6 text-white group-hover:scale-110 transition-transform" />
        <span className="absolute -top-1 -right-1 w-5 h-5 bg-red-500 rounded-full flex items-center justify-center text-white text-[10px] font-bold">
          AI
        </span>
      </button>
    );
  }

  // ── Chat window ──────────────────────────────────────────────────────────────
  return (
    <div
      className={`fixed bottom-6 right-6 bg-white rounded-2xl shadow-2xl z-50 flex flex-col transition-all duration-200 border border-gray-100 ${
        isMinimized ? 'w-80 h-14' : 'w-96 h-[580px]'
      }`}
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 bg-gradient-to-r from-violet-600 to-blue-600 rounded-t-2xl flex-shrink-0">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 bg-white/20 rounded-full flex items-center justify-center">
            <Sparkles className="w-4 h-4 text-white" />
          </div>
          <div>
            <p className="text-white font-semibold text-sm leading-none">AI Assistant</p>
            <p className="text-white/70 text-xs mt-0.5">Online</p>
          </div>
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={() => setIsMinimized((v) => !v)}
            className="p-1.5 hover:bg-white/20 rounded-lg transition-colors"
            aria-label={isMinimized ? 'Expand' : 'Minimize'}
          >
            {isMinimized ? (
              <Maximize2 className="w-4 h-4 text-white" />
            ) : (
              <Minimize2 className="w-4 h-4 text-white" />
            )}
          </button>
          <button
            onClick={() => { setIsOpen(false); setMessages([]); }}
            className="p-1.5 hover:bg-white/20 rounded-lg transition-colors"
            aria-label="Close"
          >
            <X className="w-4 h-4 text-white" />
          </button>
        </div>
      </div>

      {!isMinimized && (
        <>
          {/* Messages */}
          <div className="flex-1 overflow-y-auto px-4 py-3 space-y-3 min-h-0">
            {messages.map((msg, i) => (
              <div key={i}>
                <div className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
                  <div
                    className={`max-w-[80%] rounded-2xl px-4 py-2.5 text-sm whitespace-pre-wrap ${
                      msg.role === 'user'
                        ? 'bg-gradient-to-r from-violet-600 to-blue-600 text-white'
                        : 'bg-gray-100 text-gray-800'
                    }`}
                  >
                    {msg.content}
                  </div>
                </div>

                {/* Action buttons */}
                {msg.actions && msg.actions.length > 0 && (
                  <div className="flex flex-wrap gap-2 mt-2 ml-1">
                    {msg.actions.map((action, j) =>
                      action.url ? (
                        <Link
                          key={j}
                          href={action.url}
                          className="px-3 py-1.5 bg-violet-50 text-violet-700 rounded-full text-xs font-medium hover:bg-violet-100 transition-colors border border-violet-200"
                        >
                          {action.label}
                        </Link>
                      ) : (
                        <button
                          key={j}
                          className="px-3 py-1.5 bg-violet-50 text-violet-700 rounded-full text-xs font-medium hover:bg-violet-100 transition-colors border border-violet-200"
                        >
                          {action.label}
                        </button>
                      )
                    )}
                  </div>
                )}
              </div>
            ))}

            {/* Typing indicator */}
            {isTyping && (
              <div className="flex justify-start">
                <div className="bg-gray-100 rounded-2xl px-4 py-3">
                  <div className="flex gap-1">
                    {[0, 150, 300].map((delay) => (
                      <div
                        key={delay}
                        className="w-2 h-2 bg-gray-400 rounded-full animate-bounce"
                        style={{ animationDelay: `${delay}ms` }}
                      />
                    ))}
                  </div>
                </div>
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>

          {/* Input */}
          <div className="px-4 py-3 border-t border-gray-100 flex-shrink-0">
            <div className="flex items-center gap-2">
              <input
                type="text"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={handleKey}
                placeholder="Ask me anything..."
                disabled={isTyping}
                className="flex-1 px-4 py-2 text-sm border border-gray-200 rounded-full focus:outline-none focus:ring-2 focus:ring-violet-400 disabled:opacity-50"
              />
              <button
                onClick={handleSend}
                disabled={!input.trim() || isTyping}
                className="w-9 h-9 flex items-center justify-center bg-gradient-to-r from-violet-600 to-blue-600 text-white rounded-full hover:shadow-md transition-all disabled:opacity-40 disabled:cursor-not-allowed flex-shrink-0"
                aria-label="Send"
              >
                <Send className="w-4 h-4" />
              </button>
            </div>
            <p className="text-center text-[10px] text-gray-400 mt-1.5">
              Powered by Mnbarh AI
            </p>
          </div>
        </>
      )}
    </div>
  );
}
