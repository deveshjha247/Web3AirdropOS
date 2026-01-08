'use client'

import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import {
  User,
  Key,
  Shield,
  Bell,
  Database,
  Zap,
  Globe,
  Save,
  Eye,
  EyeOff,
} from 'lucide-react'

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState('profile')
  const [showApiKey, setShowApiKey] = useState(false)
  
  const [settings, setSettings] = useState({
    // Profile
    email: 'user@example.com',
    username: 'web3user',
    
    // API Keys
    openaiKey: 'sk-...',
    alchemyKey: 'alk_...',
    heliusKey: 'hel_...',
    
    // Notifications
    emailNotifications: true,
    browserNotifications: true,
    taskAlerts: true,
    campaignAlerts: true,
    
    // Automation
    maxConcurrentTasks: 5,
    taskDelay: 2000,
    retryOnFail: true,
    maxRetries: 3,
    
    // Browser
    defaultProxy: 'auto',
    headlessMode: false,
    screenshotOnError: true,
  })

  const tabs = [
    { id: 'profile', label: 'Profile', icon: User },
    { id: 'api', label: 'API Keys', icon: Key },
    { id: 'security', label: 'Security', icon: Shield },
    { id: 'notifications', label: 'Notifications', icon: Bell },
    { id: 'automation', label: 'Automation', icon: Zap },
    { id: 'browser', label: 'Browser', icon: Globe },
  ]

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold">Settings</h1>
        <p className="text-muted-foreground">
          Configure your Web3AirdropOS preferences
        </p>
      </div>

      <div className="flex gap-6">
        {/* Sidebar */}
        <div className="w-64 space-y-1">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg text-sm font-medium transition-colors ${
                activeTab === tab.id
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:bg-muted hover:text-foreground'
              }`}
            >
              <tab.icon className="h-5 w-5" />
              {tab.label}
            </button>
          ))}
        </div>

        {/* Content */}
        <div className="flex-1">
          {activeTab === 'profile' && (
            <Card>
              <CardHeader>
                <CardTitle>Profile Settings</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-2">Email</label>
                  <input
                    type="email"
                    value={settings.email}
                    onChange={(e) =>
                      setSettings({ ...settings, email: e.target.value })
                    }
                    className="w-full h-10 px-4 rounded-lg bg-muted border border-input text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">Username</label>
                  <input
                    type="text"
                    value={settings.username}
                    onChange={(e) =>
                      setSettings({ ...settings, username: e.target.value })
                    }
                    className="w-full h-10 px-4 rounded-lg bg-muted border border-input text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
                <Button>
                  <Save className="h-4 w-4 mr-2" />
                  Save Changes
                </Button>
              </CardContent>
            </Card>
          )}

          {activeTab === 'api' && (
            <Card>
              <CardHeader>
                <CardTitle>API Keys</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-2">
                    OpenAI API Key
                  </label>
                  <div className="flex gap-2">
                    <input
                      type={showApiKey ? 'text' : 'password'}
                      value={settings.openaiKey}
                      onChange={(e) =>
                        setSettings({ ...settings, openaiKey: e.target.value })
                      }
                      className="flex-1 h-10 px-4 rounded-lg bg-muted border border-input text-sm focus:outline-none focus:ring-2 focus:ring-primary font-mono"
                    />
                    <Button
                      variant="outline"
                      size="icon"
                      onClick={() => setShowApiKey(!showApiKey)}
                    >
                      {showApiKey ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                    </Button>
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">
                    Alchemy API Key (EVM)
                  </label>
                  <input
                    type="password"
                    value={settings.alchemyKey}
                    onChange={(e) =>
                      setSettings({ ...settings, alchemyKey: e.target.value })
                    }
                    className="w-full h-10 px-4 rounded-lg bg-muted border border-input text-sm focus:outline-none focus:ring-2 focus:ring-primary font-mono"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">
                    Helius API Key (Solana)
                  </label>
                  <input
                    type="password"
                    value={settings.heliusKey}
                    onChange={(e) =>
                      setSettings({ ...settings, heliusKey: e.target.value })
                    }
                    className="w-full h-10 px-4 rounded-lg bg-muted border border-input text-sm focus:outline-none focus:ring-2 focus:ring-primary font-mono"
                  />
                </div>
                <Button>
                  <Save className="h-4 w-4 mr-2" />
                  Save API Keys
                </Button>
              </CardContent>
            </Card>
          )}

          {activeTab === 'security' && (
            <Card>
              <CardHeader>
                <CardTitle>Security Settings</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-2">
                    Current Password
                  </label>
                  <input
                    type="password"
                    className="w-full h-10 px-4 rounded-lg bg-muted border border-input text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">
                    New Password
                  </label>
                  <input
                    type="password"
                    className="w-full h-10 px-4 rounded-lg bg-muted border border-input text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">
                    Confirm New Password
                  </label>
                  <input
                    type="password"
                    className="w-full h-10 px-4 rounded-lg bg-muted border border-input text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
                <Button>
                  <Shield className="h-4 w-4 mr-2" />
                  Update Password
                </Button>
              </CardContent>
            </Card>
          )}

          {activeTab === 'notifications' && (
            <Card>
              <CardHeader>
                <CardTitle>Notification Preferences</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                {[
                  { key: 'emailNotifications', label: 'Email Notifications' },
                  { key: 'browserNotifications', label: 'Browser Notifications' },
                  { key: 'taskAlerts', label: 'Task Completion Alerts' },
                  { key: 'campaignAlerts', label: 'Campaign Updates' },
                ].map((item) => (
                  <div
                    key={item.key}
                    className="flex items-center justify-between py-2"
                  >
                    <span className="text-sm font-medium">{item.label}</span>
                    <button
                      onClick={() =>
                        setSettings({
                          ...settings,
                          [item.key]: !settings[item.key as keyof typeof settings],
                        })
                      }
                      className={`w-12 h-6 rounded-full transition-colors ${
                        settings[item.key as keyof typeof settings]
                          ? 'bg-primary'
                          : 'bg-muted'
                      }`}
                    >
                      <div
                        className={`w-5 h-5 rounded-full bg-white shadow transition-transform ${
                          settings[item.key as keyof typeof settings]
                            ? 'translate-x-6'
                            : 'translate-x-0.5'
                        }`}
                      />
                    </button>
                  </div>
                ))}
              </CardContent>
            </Card>
          )}

          {activeTab === 'automation' && (
            <Card>
              <CardHeader>
                <CardTitle>Automation Settings</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-2">
                    Max Concurrent Tasks
                  </label>
                  <input
                    type="number"
                    value={settings.maxConcurrentTasks}
                    onChange={(e) =>
                      setSettings({
                        ...settings,
                        maxConcurrentTasks: parseInt(e.target.value),
                      })
                    }
                    className="w-full h-10 px-4 rounded-lg bg-muted border border-input text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">
                    Task Delay (ms)
                  </label>
                  <input
                    type="number"
                    value={settings.taskDelay}
                    onChange={(e) =>
                      setSettings({
                        ...settings,
                        taskDelay: parseInt(e.target.value),
                      })
                    }
                    className="w-full h-10 px-4 rounded-lg bg-muted border border-input text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">
                    Max Retries on Failure
                  </label>
                  <input
                    type="number"
                    value={settings.maxRetries}
                    onChange={(e) =>
                      setSettings({
                        ...settings,
                        maxRetries: parseInt(e.target.value),
                      })
                    }
                    className="w-full h-10 px-4 rounded-lg bg-muted border border-input text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
                <div className="flex items-center justify-between py-2">
                  <span className="text-sm font-medium">Retry on Failure</span>
                  <button
                    onClick={() =>
                      setSettings({
                        ...settings,
                        retryOnFail: !settings.retryOnFail,
                      })
                    }
                    className={`w-12 h-6 rounded-full transition-colors ${
                      settings.retryOnFail ? 'bg-primary' : 'bg-muted'
                    }`}
                  >
                    <div
                      className={`w-5 h-5 rounded-full bg-white shadow transition-transform ${
                        settings.retryOnFail
                          ? 'translate-x-6'
                          : 'translate-x-0.5'
                      }`}
                    />
                  </button>
                </div>
                <Button>
                  <Save className="h-4 w-4 mr-2" />
                  Save Settings
                </Button>
              </CardContent>
            </Card>
          )}

          {activeTab === 'browser' && (
            <Card>
              <CardHeader>
                <CardTitle>Browser Settings</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <label className="block text-sm font-medium mb-2">
                    Default Proxy
                  </label>
                  <select
                    value={settings.defaultProxy}
                    onChange={(e) =>
                      setSettings({ ...settings, defaultProxy: e.target.value })
                    }
                    className="w-full h-10 px-4 rounded-lg bg-muted border border-input text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  >
                    <option value="auto">Auto-assign</option>
                    <option value="none">No Proxy</option>
                    <option value="us-1">US Proxy 1</option>
                    <option value="eu-1">EU Proxy 1</option>
                  </select>
                </div>
                <div className="flex items-center justify-between py-2">
                  <div>
                    <span className="text-sm font-medium">Headless Mode</span>
                    <p className="text-xs text-muted-foreground">
                      Run browser without visible UI
                    </p>
                  </div>
                  <button
                    onClick={() =>
                      setSettings({
                        ...settings,
                        headlessMode: !settings.headlessMode,
                      })
                    }
                    className={`w-12 h-6 rounded-full transition-colors ${
                      settings.headlessMode ? 'bg-primary' : 'bg-muted'
                    }`}
                  >
                    <div
                      className={`w-5 h-5 rounded-full bg-white shadow transition-transform ${
                        settings.headlessMode
                          ? 'translate-x-6'
                          : 'translate-x-0.5'
                      }`}
                    />
                  </button>
                </div>
                <div className="flex items-center justify-between py-2">
                  <div>
                    <span className="text-sm font-medium">Screenshot on Error</span>
                    <p className="text-xs text-muted-foreground">
                      Capture screenshot when task fails
                    </p>
                  </div>
                  <button
                    onClick={() =>
                      setSettings({
                        ...settings,
                        screenshotOnError: !settings.screenshotOnError,
                      })
                    }
                    className={`w-12 h-6 rounded-full transition-colors ${
                      settings.screenshotOnError ? 'bg-primary' : 'bg-muted'
                    }`}
                  >
                    <div
                      className={`w-5 h-5 rounded-full bg-white shadow transition-transform ${
                        settings.screenshotOnError
                          ? 'translate-x-6'
                          : 'translate-x-0.5'
                      }`}
                    />
                  </button>
                </div>
                <Button>
                  <Save className="h-4 w-4 mr-2" />
                  Save Settings
                </Button>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  )
}
