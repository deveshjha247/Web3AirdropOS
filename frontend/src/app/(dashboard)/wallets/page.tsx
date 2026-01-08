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
      <div className="flex items-center justify-between animate-fade-in">
        <div>
          <h1 className="text-3xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent">
            Wallets
          </h1>
          <p className="text-muted-foreground mt-1">
            Manage your EVM and Solana wallets
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" className="hover-lift">
            <RefreshCw className="h-4 w-4 mr-2" />
            Sync All
          </Button>
          <Button className="hover-lift">
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
        {filteredWallets.length === 0 ? (
          <div className="col-span-full">
            <Card className="border-dashed animate-fade-in">
              <CardContent className="flex flex-col items-center justify-center py-12 text-center">
                <Wallet className="h-12 w-12 text-muted-foreground mb-4 opacity-50" />
                <h3 className="text-lg font-semibold mb-2">No wallets found</h3>
                <p className="text-sm text-muted-foreground mb-4">
                  {searchQuery || selectedGroup
                    ? 'Try adjusting your filters'
                    : 'Get started by adding your first wallet'}
                </p>
                {!searchQuery && !selectedGroup && (
                  <Button>
                    <Plus className="h-4 w-4 mr-2" />
                    Add Wallet
                  </Button>
                )}
              </CardContent>
            </Card>
          </div>
        ) : (
          <>
            {filteredWallets.map((wallet, index) => (
              <Card
                key={wallet.id}
                className="hover-lift animate-slide-up border-border/50 hover:border-primary/50 transition-all duration-300"
                style={{ animationDelay: `${index * 50}ms` }}
              >
                <CardHeader className="flex flex-row items-start justify-between pb-3">
                  <div className="flex-1 min-w-0">
                    <CardTitle className="text-base font-semibold truncate">
                      {wallet.name}
                    </CardTitle>
                    <div className="flex items-center gap-2 mt-2">
                      <span
                        className={`px-2.5 py-1 rounded-md text-xs font-medium ${
                          chainColors[wallet.chain]
                        }`}
                      >
                        {wallet.chain}
                      </span>
                    </div>
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 hover:bg-muted transition-all"
                  >
                    <MoreVertical className="h-4 w-4" />
                  </Button>
                </CardHeader>
                <CardContent>
                  <div className="space-y-4">
                    {/* Address */}
                    <div className="flex items-center justify-between p-2 rounded-lg bg-muted/50">
                      <code className="text-xs font-mono text-muted-foreground truncate flex-1">
                        {shortenAddress(wallet.address)}
                      </code>
                      <div className="flex gap-1 ml-2">
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-7 w-7 hover:bg-primary/10 hover:text-primary transition-all"
                          title="Copy Address"
                        >
                          <Copy className="h-3.5 w-3.5" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-7 w-7 hover:bg-primary/10 hover:text-primary transition-all"
                          title="View on Explorer"
                        >
                          <ExternalLink className="h-3.5 w-3.5" />
                        </Button>
                      </div>
                    </div>

                    {/* Balance */}
                    <div>
                      <p className="text-xs text-muted-foreground mb-1">Balance</p>
                      <div className="text-2xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent">
                        {wallet.balance}
                      </div>
                    </div>

                    {/* Tags */}
                    {wallet.tags.length > 0 && (
                      <div className="flex flex-wrap gap-1.5">
                        {wallet.tags.map((tag) => (
                          <span
                            key={tag}
                            className="px-2 py-1 rounded-md text-xs bg-muted/50 text-muted-foreground flex items-center gap-1"
                          >
                            <Tag className="h-3 w-3" />
                            {tag}
                          </span>
                        ))}
                      </div>
                    )}

                    {/* Actions */}
                    <div className="flex gap-2 pt-3 border-t border-border">
                      <Button
                        variant="outline"
                        size="sm"
                        className="flex-1 hover:bg-primary/10 hover:border-primary/50 transition-all"
                      >
                        <RefreshCw className="h-3 w-3 mr-1.5" />
                        Sync
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        className="flex-1 hover:bg-primary/10 hover:border-primary/50 transition-all"
                      >
                        View
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}

            {/* Add Wallet Card */}
            <Card className="border-dashed hover-lift animate-slide-up cursor-pointer group hover:border-primary/50 transition-all duration-300">
              <CardContent className="flex flex-col items-center justify-center h-full min-h-[200px] text-muted-foreground group-hover:text-primary transition-colors">
                <div className="h-12 w-12 rounded-full bg-primary/10 flex items-center justify-center mb-3 group-hover:bg-primary/20 transition-colors">
                  <Plus className="h-6 w-6" />
                </div>
                <span className="font-medium">Add New Wallet</span>
                <span className="text-xs mt-1">Import or create wallet</span>
              </CardContent>
            </Card>
          </>
        )}
      </div>
    </div>
  )
}
