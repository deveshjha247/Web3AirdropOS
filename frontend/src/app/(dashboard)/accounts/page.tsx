'use client'

import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import {
  Plus,
  Search,
  MoreVertical,
  Wallet,
  ExternalLink,
  CheckCircle,
  XCircle,
  RefreshCw,
} from 'lucide-react'

// Mock data
const accounts = [
  {
    id: '1',
    platform: 'farcaster',
    username: 'cryptobuilder.eth',
    displayName: 'Crypto Builder',
    status: 'active',
    followers: 1250,
    following: 340,
    linkedWallets: 2,
    lastSync: '2 hours ago',
  },
  {
    id: '2',
    platform: 'twitter',
    username: '@web3enthusiast',
    displayName: 'Web3 Enthusiast',
    status: 'active',
    followers: 5420,
    following: 890,
    linkedWallets: 1,
    lastSync: '1 hour ago',
  },
  {
    id: '3',
    platform: 'discord',
    username: 'defi_trader#1234',
    displayName: 'DeFi Trader',
    status: 'active',
    followers: 0,
    following: 0,
    linkedWallets: 3,
    lastSync: '30 mins ago',
  },
  {
    id: '4',
    platform: 'telegram',
    username: '@airdrop_hunter',
    displayName: 'Airdrop Hunter',
    status: 'inactive',
    followers: 0,
    following: 0,
    linkedWallets: 0,
    lastSync: 'Never',
  },
]

const platformColors: Record<string, { bg: string; text: string; icon: string }> = {
  farcaster: { bg: 'bg-purple-500/20', text: 'text-purple-400', icon: '🟣' },
  twitter: { bg: 'bg-sky-500/20', text: 'text-sky-400', icon: '𝕏' },
  discord: { bg: 'bg-indigo-500/20', text: 'text-indigo-400', icon: '💬' },
  telegram: { bg: 'bg-blue-500/20', text: 'text-blue-400', icon: '✈️' },
}

export default function AccountsPage() {
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedPlatform, setSelectedPlatform] = useState<string | null>(null)

  const filteredAccounts = accounts.filter((account) => {
    const matchesSearch =
      account.username.toLowerCase().includes(searchQuery.toLowerCase()) ||
      account.displayName.toLowerCase().includes(searchQuery.toLowerCase())
    const matchesPlatform =
      !selectedPlatform || account.platform === selectedPlatform
    return matchesSearch && matchesPlatform
  })

  const platforms = [...new Set(accounts.map((a) => a.platform))]

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Accounts</h1>
          <p className="text-muted-foreground">
            Manage your social platform accounts
          </p>
        </div>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          Link Account
        </Button>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search accounts..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full h-10 pl-10 pr-4 rounded-lg bg-muted border border-input text-sm focus:outline-none focus:ring-2 focus:ring-primary"
          />
        </div>
        <div className="flex gap-2">
          {platforms.map((platform) => (
            <Button
              key={platform}
              variant={selectedPlatform === platform ? 'default' : 'outline'}
              size="sm"
              onClick={() =>
                setSelectedPlatform(
                  selectedPlatform === platform ? null : platform
                )
              }
              className="capitalize"
            >
              <span className="mr-1">{platformColors[platform]?.icon}</span>
              {platform}
            </Button>
          ))}
        </div>
      </div>

      {/* Accounts Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {filteredAccounts.map((account) => (
          <Card
            key={account.id}
            className="hover:border-primary/50 transition-colors"
          >
            <CardHeader className="flex flex-row items-start justify-between pb-2">
              <div className="flex items-center gap-3">
                <div
                  className={`h-12 w-12 rounded-full flex items-center justify-center text-2xl ${
                    platformColors[account.platform]?.bg
                  }`}
                >
                  {platformColors[account.platform]?.icon}
                </div>
                <div>
                  <CardTitle className="text-base font-medium">
                    {account.displayName}
                  </CardTitle>
                  <p className="text-sm text-muted-foreground">
                    {account.username}
                  </p>
                </div>
              </div>
              <Button variant="ghost" size="icon" className="h-8 w-8">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {/* Status & Platform */}
                <div className="flex items-center justify-between">
                  <span
                    className={`px-2 py-0.5 rounded text-xs capitalize ${
                      platformColors[account.platform]?.bg
                    } ${platformColors[account.platform]?.text}`}
                  >
                    {account.platform}
                  </span>
                  <div className="flex items-center gap-1">
                    {account.status === 'active' ? (
                      <CheckCircle className="h-4 w-4 text-green-400" />
                    ) : (
                      <XCircle className="h-4 w-4 text-red-400" />
                    )}
                    <span
                      className={`text-xs ${
                        account.status === 'active'
                          ? 'text-green-400'
                          : 'text-red-400'
                      }`}
                    >
                      {account.status}
                    </span>
                  </div>
                </div>

                {/* Stats */}
                {account.followers > 0 && (
                  <div className="flex gap-4 text-sm">
                    <div>
                      <span className="font-medium">{account.followers}</span>
                      <span className="text-muted-foreground ml-1">
                        followers
                      </span>
                    </div>
                    <div>
                      <span className="font-medium">{account.following}</span>
                      <span className="text-muted-foreground ml-1">
                        following
                      </span>
                    </div>
                  </div>
                )}

                {/* Linked Wallets */}
                <div className="flex items-center gap-2 text-sm">
                  <Wallet className="h-4 w-4 text-muted-foreground" />
                  <span>
                    {account.linkedWallets} linked wallet
                    {account.linkedWallets !== 1 ? 's' : ''}
                  </span>
                </div>

                {/* Last Sync */}
                <div className="text-xs text-muted-foreground">
                  Last synced: {account.lastSync}
                </div>

                {/* Actions */}
                <div className="flex gap-2 pt-2 border-t border-border">
                  <Button variant="outline" size="sm" className="flex-1">
                    <RefreshCw className="h-3 w-3 mr-1" />
                    Sync
                  </Button>
                  <Button variant="outline" size="sm" className="flex-1">
                    <Wallet className="h-3 w-3 mr-1" />
                    Link Wallet
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}

        {/* Add Account Card */}
        <Card className="border-dashed hover:border-primary/50 transition-colors cursor-pointer">
          <CardContent className="flex flex-col items-center justify-center h-full min-h-[200px] text-muted-foreground">
            <Plus className="h-10 w-10 mb-2" />
            <span>Link New Account</span>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
