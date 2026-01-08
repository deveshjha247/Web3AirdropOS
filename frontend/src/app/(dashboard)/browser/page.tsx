'use client'

import { useState, useEffect, useRef } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import {
  Plus,
  Maximize2,
  Minimize2,
  RefreshCw,
  ArrowLeft,
  ArrowRight,
  Home,
  Camera,
  Mouse,
  Keyboard,
  Settings,
  Play,
  Pause,
  MoreVertical,
} from 'lucide-react'

// Mock browser sessions
const sessions = [
  {
    id: '1',
    profileName: 'Airdrop Profile 1',
    currentUrl: 'https://galxe.com/quest/example',
    status: 'active',
    wallet: '0x1234...5678',
  },
  {
    id: '2',
    profileName: 'Farcaster Account',
    currentUrl: 'https://warpcast.com',
    status: 'active',
    wallet: '0xabcd...efgh',
  },
  {
    id: '3',
    profileName: 'Twitter Bot',
    currentUrl: 'https://twitter.com',
    status: 'idle',
    wallet: null,
  },
]

const profiles = [
  { id: '1', name: 'Airdrop Profile 1', proxy: 'US Proxy 1' },
  { id: '2', name: 'Farcaster Account', proxy: 'EU Proxy 2' },
  { id: '3', name: 'Twitter Bot', proxy: 'No Proxy' },
  { id: '4', name: 'Discord Account', proxy: 'US Proxy 3' },
]

export default function BrowserPage() {
  const [activeSession, setActiveSession] = useState<string | null>(sessions[0]?.id || null)
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [urlInput, setUrlInput] = useState('')
  const vncRef = useRef<HTMLDivElement>(null)

  const currentSession = sessions.find((s) => s.id === activeSession)

  useEffect(() => {
    if (currentSession) {
      setUrlInput(currentSession.currentUrl)
    }
  }, [currentSession])

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Browser Workspace</h1>
          <p className="text-muted-foreground">
            Embedded browser with manual takeover capability
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline">
            <Settings className="h-4 w-4 mr-2" />
            Profiles
          </Button>
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            New Session
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Left Sidebar - Sessions */}
        <Card className="lg:col-span-1">
          <CardHeader className="py-4">
            <CardTitle className="text-sm font-medium">Active Sessions</CardTitle>
          </CardHeader>
          <CardContent className="p-2">
            <div className="space-y-2">
              {sessions.map((session) => (
                <div
                  key={session.id}
                  className={`p-3 rounded-lg cursor-pointer transition-colors ${
                    activeSession === session.id
                      ? 'bg-primary/20 border border-primary'
                      : 'bg-muted hover:bg-muted/80'
                  }`}
                  onClick={() => setActiveSession(session.id)}
                >
                  <div className="flex items-center justify-between mb-1">
                    <span className="font-medium text-sm">
                      {session.profileName}
                    </span>
                    <div
                      className={`h-2 w-2 rounded-full ${
                        session.status === 'active'
                          ? 'bg-green-400'
                          : 'bg-yellow-400'
                      }`}
                    />
                  </div>
                  <p className="text-xs text-muted-foreground truncate">
                    {session.currentUrl}
                  </p>
                  {session.wallet && (
                    <p className="text-xs text-muted-foreground mt-1">
                      Wallet: {session.wallet}
                    </p>
                  )}
                </div>
              ))}
            </div>

            <div className="mt-4 pt-4 border-t border-border">
              <p className="text-xs font-medium text-muted-foreground mb-2">
                Available Profiles
              </p>
              {profiles.slice(0, 3).map((profile) => (
                <div
                  key={profile.id}
                  className="flex items-center justify-between py-2 text-sm"
                >
                  <span>{profile.name}</span>
                  <Button variant="ghost" size="sm">
                    <Play className="h-3 w-3" />
                  </Button>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Main Browser Area */}
        <div className="lg:col-span-3 space-y-4">
          {/* Browser Toolbar */}
          <Card className="p-2">
            <div className="flex items-center gap-2">
              {/* Navigation */}
              <div className="flex gap-1">
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <ArrowLeft className="h-4 w-4" />
                </Button>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <ArrowRight className="h-4 w-4" />
                </Button>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <RefreshCw className="h-4 w-4" />
                </Button>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <Home className="h-4 w-4" />
                </Button>
              </div>

              {/* URL Bar */}
              <div className="flex-1">
                <input
                  type="text"
                  value={urlInput}
                  onChange={(e) => setUrlInput(e.target.value)}
                  placeholder="Enter URL..."
                  className="w-full h-8 px-3 rounded bg-muted border border-input text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>

              {/* Actions */}
              <div className="flex gap-1">
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <Camera className="h-4 w-4" />
                </Button>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <Mouse className="h-4 w-4" />
                </Button>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <Keyboard className="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8"
                  onClick={() => setIsFullscreen(!isFullscreen)}
                >
                  {isFullscreen ? (
                    <Minimize2 className="h-4 w-4" />
                  ) : (
                    <Maximize2 className="h-4 w-4" />
                  )}
                </Button>
              </div>
            </div>
          </Card>

          {/* Browser Viewport (VNC) */}
          <Card className={`overflow-hidden ${isFullscreen ? 'fixed inset-4 z-50' : ''}`}>
            <div
              ref={vncRef}
              className="vnc-container relative"
              style={{ height: isFullscreen ? '100%' : '600px' }}
            >
              {/* Placeholder for VNC viewer */}
              <div className="absolute inset-0 flex items-center justify-center bg-gradient-to-br from-gray-900 to-gray-800">
                <div className="text-center">
                  <div className="w-16 h-16 rounded-full bg-primary/20 flex items-center justify-center mx-auto mb-4">
                    <Play className="h-8 w-8 text-primary" />
                  </div>
                  <h3 className="text-lg font-medium mb-2">
                    {currentSession
                      ? `Session: ${currentSession.profileName}`
                      : 'No Active Session'}
                  </h3>
                  <p className="text-sm text-muted-foreground mb-4">
                    {currentSession
                      ? 'Click to take manual control'
                      : 'Start a session to begin'}
                  </p>
                  {currentSession && (
                    <div className="flex gap-2 justify-center">
                      <Button variant="outline" size="sm">
                        <Mouse className="h-4 w-4 mr-2" />
                        Take Control
                      </Button>
                      <Button variant="outline" size="sm">
                        <Pause className="h-4 w-4 mr-2" />
                        Pause Automation
                      </Button>
                    </div>
                  )}
                </div>
              </div>

              {/* Status Bar */}
              <div className="absolute bottom-0 left-0 right-0 h-8 bg-black/80 flex items-center px-4 text-xs">
                <div className="flex items-center gap-4">
                  <span className="flex items-center gap-1">
                    <div className="h-2 w-2 rounded-full bg-green-400" />
                    Connected
                  </span>
                  <span className="text-muted-foreground">
                    Resolution: 1920x1080
                  </span>
                  <span className="text-muted-foreground">
                    Latency: 45ms
                  </span>
                </div>
                <div className="ml-auto flex items-center gap-2">
                  <span className="text-muted-foreground">
                    {currentSession?.currentUrl}
                  </span>
                </div>
              </div>
            </div>
          </Card>

          {/* Quick Actions */}
          <div className="flex gap-2">
            <Button variant="outline" size="sm">
              Connect Wallet
            </Button>
            <Button variant="outline" size="sm">
              Sign Transaction
            </Button>
            <Button variant="outline" size="sm">
              Clear Cookies
            </Button>
            <Button variant="outline" size="sm">
              Switch Profile
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
