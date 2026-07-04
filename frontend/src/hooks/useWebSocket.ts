import { useEffect, useRef, useCallback } from 'react'

type MessageHandler = (type: string, payload: unknown) => void

export function useWebSocket(onMessage: MessageHandler) {
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimer = useRef<ReturnType<typeof setTimeout>>()

  const connect = useCallback(() => {
    const token = localStorage.getItem('access_token')
    if (!token) return

    const wsBase = import.meta.env.VITE_WS_URL ?? 'ws://localhost:8080'
    const ws = new WebSocket(`${wsBase}/ws?token=${token}`)
    wsRef.current = ws

    ws.onopen = () => {
      console.log('WebSocket connected')
    }

    ws.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data) as { type: string; payload: unknown }
        onMessage(msg.type, msg.payload)
      } catch { /* ignore malformed */ }
    }

    ws.onclose = (e) => {
      console.log('WebSocket closed:', e.code, e.reason)
      reconnectTimer.current = setTimeout(connect, 5000)
    }

    ws.onerror = (e) => {
      console.error('WebSocket error:', e)
      ws.close()
    }
  }, [onMessage])

  useEffect(() => {
    connect()
    return () => {
      clearTimeout(reconnectTimer.current)
      wsRef.current?.close()
    }
  }, [connect])
}
