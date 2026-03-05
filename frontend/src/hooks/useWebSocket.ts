'use client'
import { useEffect, useRef, useState, useCallback } from 'react'

const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080'

export function useChatWebSocket(conversationId: string | null) {
  const ws = useRef<WebSocket | null>(null)
  const [messages, setMessages] = useState<string[]>([])
  const [connected, setConnected] = useState(false)

  useEffect(() => {
    if (!conversationId) return
    const token = localStorage.getItem('token')
    ws.current = new WebSocket(`${WS_URL}/ws/chat/${conversationId}?token=${token}`)

    ws.current.onopen = () => setConnected(true)
    ws.current.onclose = () => setConnected(false)
    ws.current.onmessage = (e) => {
      setMessages((prev) => [...prev, e.data])
    }

    return () => ws.current?.close()
  }, [conversationId])

  const send = useCallback((msg: string) => {
    if (ws.current?.readyState === WebSocket.OPEN) {
      ws.current.send(msg)
    }
  }, [])

  return { messages, connected, send }
}

export function useAuctionWebSocket(auctionId: string | null, onBid?: (data: any) => void) {
  const ws = useRef<WebSocket | null>(null)
  const [connected, setConnected] = useState(false)

  useEffect(() => {
    if (!auctionId) return
    ws.current = new WebSocket(`${WS_URL}/ws/auctions/${auctionId}`)
    ws.current.onopen = () => setConnected(true)
    ws.current.onclose = () => setConnected(false)
    ws.current.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data)
        onBid?.(data)
      } catch {}
    }
    return () => ws.current?.close()
  }, [auctionId, onBid])

  return { connected }
}
