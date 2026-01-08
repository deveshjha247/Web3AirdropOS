'use client'

import { useEffect, useRef, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Trash2, Download, Maximize2 } from 'lucide-react'

interface TerminalLine {
  id: string
  timestamp: string
  level: 'info' | 'success' | 'error' | 'warning'
  message: string
}

export function Terminal() {
  const [lines, setLines] = useState<TerminalLine[]>([
    {
      id: '1',
      timestamp: new Date().toISOString(),
      level: 'info',
      message: 'Web3AirdropOS Terminal initialized',
    },
    {
      id: '2',
      timestamp: new Date().toISOString(),
      level: 'success',
      message: 'Connected to backend service',
    },
    {
      id: '3',
      timestamp: new Date().toISOString(),
      level: 'info',
      message: 'Listening for real-time updates...',
    },
  ])
  const terminalRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom
  useEffect(() => {
    if (terminalRef.current) {
      terminalRef.current.scrollTop = terminalRef.current.scrollHeight
    }
  }, [lines])

  // WebSocket connection for real-time logs
  useEffect(() => {
    const wsUrl = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080'
    let ws: WebSocket | null = null

    try {
      ws = new WebSocket(`${wsUrl}/ws`)

      ws.onopen = () => {
        setLines((prev) => [
          ...prev,
          {
            id: crypto.randomUUID(),
            timestamp: new Date().toISOString(),
            level: 'success',
            message: 'WebSocket connected',
          },
        ])
      }

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          if (data.type === 'terminal') {
            setLines((prev) => [
              ...prev,
              {
                id: crypto.randomUUID(),
                timestamp: data.timestamp || new Date().toISOString(),
                level: data.level || 'info',
                message: data.message,
              },
            ])
          }
        } catch (e) {
          // Handle non-JSON messages
          setLines((prev) => [
            ...prev,
            {
              id: crypto.randomUUID(),
              timestamp: new Date().toISOString(),
              level: 'info',
              message: event.data,
            },
          ])
        }
      }

      ws.onerror = () => {
        setLines((prev) => [
          ...prev,
          {
            id: crypto.randomUUID(),
            timestamp: new Date().toISOString(),
            level: 'warning',
            message: 'WebSocket connection error - using mock data',
          },
        ])
      }

      ws.onclose = () => {
        setLines((prev) => [
          ...prev,
          {
            id: crypto.randomUUID(),
            timestamp: new Date().toISOString(),
            level: 'warning',
            message: 'WebSocket disconnected',
          },
        ])
      }
    } catch (e) {
      // WebSocket not available
    }

    return () => {
      ws?.close()
    }
  }, [])

  const clearTerminal = () => {
    setLines([
      {
        id: crypto.randomUUID(),
        timestamp: new Date().toISOString(),
        level: 'info',
        message: 'Terminal cleared',
      },
    ])
  }

  const formatTime = (timestamp: string) => {
    return new Date(timestamp).toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  }

  const levelColors = {
    info: 'text-blue-400',
    success: 'text-green-400',
    error: 'text-red-400',
    warning: 'text-yellow-400',
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between py-3">
        <CardTitle className="text-lg">Terminal</CardTitle>
        <div className="flex items-center gap-1">
          <Button variant="ghost" size="icon" className="h-8 w-8">
            <Maximize2 className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="icon" className="h-8 w-8">
            <Download className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={clearTerminal}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </CardHeader>
      <CardContent className="p-0">
        <div
          ref={terminalRef}
          className="terminal h-64 overflow-auto bg-black/50 p-4 rounded-b-lg"
        >
          {lines.map((line) => (
            <div key={line.id} className={`terminal-line ${line.level}`}>
              <span className="text-muted-foreground">
                [{formatTime(line.timestamp)}]
              </span>{' '}
              <span className={levelColors[line.level]}>{line.message}</span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
