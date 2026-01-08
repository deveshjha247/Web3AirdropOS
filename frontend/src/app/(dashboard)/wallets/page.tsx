'use client'

import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import {
  Plus,
  Search,
  Filter,
  MoreVertical,
  Copy,
  Trash2,
  RefreshCw,
  ExternalLink,
  Tag,
  Folder,
} from 'lucide-react'

// Mock data
const wallets = [
  {
    id: '1',
    name: 'Main Wallet',
    address: '0x1234567890abcdef1234567890abcdef12345678',
    chain: 'ethereum',
    balance: '1.5 ETH',
    tags: ['main', 'active'],
    group: 'Trading',
  },
  {
    id: '2',
    name: 'Airdrop Wallet 1',
    address: '0xabcdef1234567890abcdef1234567890abcdef12',
    chain: 'ethereum',
    balance: '0.3 ETH',
    tags: ['airdrop'],
    group: 'Airdrops',
  },
  {
    id: '3',
    name: 'Solana Main',
    address: 'So11111111111111111111111111111111111111112',
    chain: 'solana',
    balance: '25 SOL',
    tags: ['main'],
    group: 'Trading',
  },
  {
    id: '4',
    name: 'Airdrop Wallet 2',
    address: '0x567890abcdef1234567890abcdef1234567890ab',
    chain: 'polygon',
    balance: '150 MATIC',
    tags: ['airdrop', 'polygon'],
    group: 'Airdrops',
  },
]

const chainColors: Record<string, string> = {
  ethereum: 'bg-blue-500/20 text-blue-400',
  solana: 'bg-purple-500/20 text-purple-400',
  polygon: 'bg-violet-500/20 text-violet-400',
  arbitrum: 'bg-sky-500/20 text-sky-400',
  optimism: 'bg-red-500/20 text-red-400',
  base: 'bg-blue-600/20 text-blue-300',
}

export default function WalletsPage() {
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedGroup, setSelectedGroup] = useState<string | null>(null)

  const filteredWallets = wallets.filter((wallet) => {
    const matchesSearch =
      wallet.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      wallet.address.toLowerCase().includes(searchQuery.toLowerCase())
    const matchesGroup = !selectedGroup || wallet.group === selectedGroup
    return matchesSearch && matchesGroup
  })

  const groups = [...new Set(wallets.map((w) => w.group))]

  const shortenAddress = (address: string) => {
    return `${address.slice(0, 6)}...${address.slice(-4)}`
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Wallets</h1>
          <p className="text-muted-foreground">
            Manage your EVM and Solana wallets
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline">
            <RefreshCw className="h-4 w-4 mr-2" />
            Sync All
          </Button>
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            Add Wallet
          </Button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search wallets..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full h-10 pl-10 pr-4 rounded-lg bg-muted border border-input text-sm focus:outline-none focus:ring-2 focus:ring-primary"
          />
        </div>
        <div className="flex gap-2">
          {groups.map((group) => (
            <Button
              key={group}
              variant={selectedGroup === group ? 'default' : 'outline'}
              size="sm"
              onClick={() =>
                setSelectedGroup(selectedGroup === group ? null : group)
              }
            >
              <Folder className="h-4 w-4 mr-1" />
              {group}
            </Button>
          ))}
        </div>
      </div>

      {/* Wallets Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {filteredWallets.map((wallet) => (
          <Card key={wallet.id} className="hover:border-primary/50 transition-colors">
            <CardHeader className="flex flex-row items-start justify-between pb-2">
              <div>
                <CardTitle className="text-base font-medium">
                  {wallet.name}
                </CardTitle>
                <div className="flex items-center gap-2 mt-1">
                  <span
                    className={`px-2 py-0.5 rounded text-xs ${
                      chainColors[wallet.chain]
                    }`}
                  >
                    {wallet.chain}
                  </span>
                </div>
              </div>
              <Button variant="ghost" size="icon" className="h-8 w-8">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {/* Address */}
                <div className="flex items-center justify-between">
                  <code className="text-sm text-muted-foreground">
                    {shortenAddress(wallet.address)}
                  </code>
                  <div className="flex gap-1">
                    <Button variant="ghost" size="icon" className="h-7 w-7">
                      <Copy className="h-3 w-3" />
                    </Button>
                    <Button variant="ghost" size="icon" className="h-7 w-7">
                      <ExternalLink className="h-3 w-3" />
                    </Button>
                  </div>
                </div>

                {/* Balance */}
                <div className="text-2xl font-bold">{wallet.balance}</div>

                {/* Tags */}
                <div className="flex flex-wrap gap-1">
                  {wallet.tags.map((tag) => (
                    <span
                      key={tag}
                      className="px-2 py-0.5 rounded-full text-xs bg-muted text-muted-foreground"
                    >
                      <Tag className="inline h-3 w-3 mr-1" />
                      {tag}
                    </span>
                  ))}
                </div>

                {/* Actions */}
                <div className="flex gap-2 pt-2 border-t border-border">
                  <Button variant="outline" size="sm" className="flex-1">
                    <RefreshCw className="h-3 w-3 mr-1" />
                    Sync
                  </Button>
                  <Button variant="outline" size="sm" className="flex-1">
                    View
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}

        {/* Add Wallet Card */}
        <Card className="border-dashed hover:border-primary/50 transition-colors cursor-pointer">
          <CardContent className="flex flex-col items-center justify-center h-full min-h-[200px] text-muted-foreground">
            <Plus className="h-10 w-10 mb-2" />
            <span>Add New Wallet</span>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
