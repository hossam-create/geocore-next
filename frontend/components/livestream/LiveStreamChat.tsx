'use client';

import { useState, useEffect, useRef } from 'react';
import { Send, MessageCircle } from 'lucide-react';

interface ChatMessage {
  id: string;
  user: string;
  text: string;
  ts: number;
  isSystem?: boolean;
}

interface LiveStreamChatProps {
  sessionId: string;
  currentUser: string;
}

let msgIdCounter = 0;

export default function LiveStreamChat({ sessionId, currentUser }: LiveStreamChatProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([
    { id: 'sys-0', user: 'System', text: 'Welcome to the live auction!', ts: Date.now(), isSystem: true },
  ]);
  const [input, setInput] = useState('');
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const sendMessage = () => {
    const text = input.trim();
    if (!text) return;
    const msg: ChatMessage = {
      id: `msg-${++msgIdCounter}`,
      user: currentUser || 'Anonymous',
      text,
      ts: Date.now(),
    };
    setMessages((prev) => [...prev, msg]);
    setInput('');
  };

  const handleKey = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  return (
    <div className="flex flex-col h-full bg-white rounded-2xl border border-gray-100 shadow-sm overflow-hidden">
      <div className="px-4 py-3 border-b border-gray-100 flex items-center gap-2">
        <MessageCircle className="w-4 h-4 text-[#0071CE]" />
        <span className="font-semibold text-sm text-gray-800">Live Chat</span>
      </div>

      <div className="flex-1 overflow-y-auto p-3 space-y-2 min-h-0">
        {messages.map((m) => (
          <div key={m.id} className={m.isSystem ? 'text-center' : 'flex gap-2'}>
            {m.isSystem ? (
              <p className="text-xs text-gray-400 bg-gray-50 px-3 py-1 rounded-full inline-block">{m.text}</p>
            ) : (
              <>
                <div className="w-6 h-6 rounded-full bg-gradient-to-br from-violet-500 to-blue-500 flex items-center justify-center flex-shrink-0 mt-0.5">
                  <span className="text-white text-[10px] font-bold">{m.user[0]?.toUpperCase()}</span>
                </div>
                <div>
                  <span className="text-[11px] font-semibold text-gray-700">{m.user}</span>
                  <p className="text-xs text-gray-600 leading-relaxed">{m.text}</p>
                </div>
              </>
            )}
          </div>
        ))}
        <div ref={bottomRef} />
      </div>

      <div className="p-3 border-t border-gray-100 flex gap-2">
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKey}
          placeholder="Say something…"
          className="flex-1 text-xs border border-gray-200 rounded-full px-3 py-2 focus:outline-none focus:ring-2 focus:ring-violet-300"
        />
        <button
          onClick={sendMessage}
          className="w-8 h-8 flex items-center justify-center bg-[#0071CE] text-white rounded-full hover:bg-[#005BA1] transition-colors flex-shrink-0"
        >
          <Send className="w-3.5 h-3.5" />
        </button>
      </div>
    </div>
  );
}
