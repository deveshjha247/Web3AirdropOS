'use client'

import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import {
  Plus,
  Search,
  Play,
  Pause,
  MoreVertical,
  ExternalLink,
  CheckCircle,
  Clock,
  AlertCircle,
  Zap,
} from 'lucide-react'

// Mock data
const campaigns = [
  {
    id: '1',
    name: 'Galxe - Layer3 Quest Season 5',
    platform: 'galxe',
    url: 'https://galxe.com/quest/...',
    status: 'running',
    progress: 75,
    walletsAssigned: 16,
    walletsCompleted: 12,
    tasksTotal: 8,
    tasksCompleted: 6,
    createdAt: '2024-01-15',
    estimatedReward: '$50-100',
  },
  {
    id: '2',
    name: 'Zealy - Community Sprint Week 12',
    platform: 'zealy',
    url: 'https://zealy.io/c/...',
    status: 'running',
    progress: 40,
    walletsAssigned: 10,
    walletsCompleted: 4,
    tasksTotal: 12,
    tasksCompleted: 5,
    createdAt: '2024-01-14',
    estimatedReward: 'XP + Role',
  },
  {
    id: '3',
    name: 'Layer3 - DeFi Discovery Path',
    platform: 'layer3',
    url: 'https://layer3.xyz/...',
    status: 'paused',
    progress: 90,
    walletsAssigned: 5,
    walletsCompleted: 4,
    tasksTotal: 5,
    tasksCompleted: 4,
    createdAt: '2024-01-10',
    estimatedReward: 'Cubes + NFT',
  },
  {
    id: '4',
    name: 'Intract - Scroll Ecosystem',
    platform: 'intract',
    url: 'https://intract.io/...',
    status: 'completed',
    progress: 100,
    walletsAssigned: 8,
    walletsCompleted: 8,
    tasksTotal: 6,
    tasksCompleted: 6,
    createdAt: '2024-01-05',
    estimatedReward: '$20-30',
  },
]

const platformStyles: Record<string, { bg: string; text: string }> = {
  galxe: { bg: 'bg-purple-500/20', text: 'text-purple-400' },
  zealy: { bg: 'bg-blue-500/20', text: 'text-blue-400' },
  layer3: { bg: 'bg-green-500/20', text: 'text-green-400' },
  intract: { bg: 'bg-orange-500/20', text: 'text-orange-400' },
}

const statusStyles: Record<string, { bg: string; text: string; icon: any }> = {
  running: { bg: 'bg-green-500/20', text: 'text-green-400', icon: Play },
  paused: { bg: 'bg-yellow-500/20', text: 'text-yellow-400', icon: Pause },
  completed: { bg: 'bg-blue-500/20', text: 'text-blue-400', icon: CheckCircle },
  failed: { bg: 'bg-red-500/20', text: 'text-red-400', icon: AlertCircle },
}

export default function CampaignsPage() {
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedPlatform, setSelectedPlatform] = useState<string | null>(null)
  const [selectedStatus, setSelectedStatus] = useState<string | null>(null)

  const filteredCampaigns = campaigns.filter((campaign) => {
    const matchesSearch = campaign.name
      .toLowerCase()
      .includes(searchQuery.toLowerCase())
    const matchesPlatform =
      !selectedPlatform || campaign.platform === selectedPlatform
    const matchesStatus = !selectedStatus || campaign.status === selectedStatus
    return matchesSearch && matchesPlatform && matchesStatus
  })

  const platforms = [...new Set(campaigns.map((c) => c.platform))]
  const statuses = [...new Set(campaigns.map((c) => c.status))]

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between animate-fade-in">
        <div>
          <h1 className="text-3xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent">
            Campaigns
          </h1>
          <p className="text-muted-foreground mt-1">
            Manage airdrop and reward campaigns
          </p>
        </div>
        <Button className="hover-lift">
          <Plus className="h-4 w-4 mr-2" />
          Add Campaign
        </Button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card className="hover-lift animate-slide-up border-green-500/20 hover:border-green-500/40 transition-all" style={{ animationDelay: '0ms' }}>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground mb-1">Active</p>
                <p className="text-3xl font-bold text-green-400">
                  {campaigns.filter((c) => c.status === 'running').length}
                </p>
              </div>
              <div className="h-12 w-12 rounded-lg bg-green-500/10 flex items-center justify-center">
                <Play className="h-6 w-6 text-green-400" />
              </div>
            </div>
          </CardContent>
        </Card>
        <Card className="hover-lift animate-slide-up border-yellow-500/20 hover:border-yellow-500/40 transition-all" style={{ animationDelay: '50ms' }}>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground mb-1">Paused</p>
                <p className="text-3xl font-bold text-yellow-400">
                  {campaigns.filter((c) => c.status === 'paused').length}
                </p>
              </div>
              <div className="h-12 w-12 rounded-lg bg-yellow-500/10 flex items-center justify-center">
                <Pause className="h-6 w-6 text-yellow-400" />
              </div>
            </div>
          </CardContent>
        </Card>
        <Card className="hover-lift animate-slide-up border-blue-500/20 hover:border-blue-500/40 transition-all" style={{ animationDelay: '100ms' }}>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground mb-1">Completed</p>
                <p className="text-3xl font-bold text-blue-400">
                  {campaigns.filter((c) => c.status === 'completed').length}
                </p>
              </div>
              <div className="h-12 w-12 rounded-lg bg-blue-500/10 flex items-center justify-center">
                <CheckCircle className="h-6 w-6 text-blue-400" />
              </div>
            </div>
          </CardContent>
        </Card>
        <Card className="hover-lift animate-slide-up border-purple-500/20 hover:border-purple-500/40 transition-all" style={{ animationDelay: '150ms' }}>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground mb-1">Total Tasks</p>
                <p className="text-3xl font-bold text-purple-400">
                  {campaigns.reduce((sum, c) => sum + c.tasksCompleted, 0)}/
                  <span className="text-lg text-muted-foreground">
                    {campaigns.reduce((sum, c) => sum + c.tasksTotal, 0)}
                  </span>
                </p>
              </div>
              <div className="h-12 w-12 rounded-lg bg-purple-500/10 flex items-center justify-center">
                <Zap className="h-6 w-6 text-purple-400" />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4 flex-wrap">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search campaigns..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full h-10 pl-10 pr-4 rounded-lg bg-muted border border-input text-sm focus:outline-none focus:ring-2 focus:ring-primary"
          />
        </div>
        <div className="flex gap-2 flex-wrap">
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
              {platform}
            </Button>
          ))}
        </div>
        <div className="flex gap-2">
          {statuses.map((status) => {
            const StatusIcon = statusStyles[status]?.icon || Clock
            return (
              <Button
                key={status}
                variant={selectedStatus === status ? 'default' : 'outline'}
                size="sm"
                onClick={() =>
                  setSelectedStatus(selectedStatus === status ? null : status)
                }
                className="capitalize"
              >
                <StatusIcon className="h-3 w-3 mr-1" />
                {status}
              </Button>
            )
          })}
        </div>
      </div>

      {/* Campaigns List */}
      <div className="space-y-4">
        {filteredCampaigns.length === 0 ? (
          <Card className="border-dashed animate-fade-in">
            <CardContent className="flex flex-col items-center justify-center py-12 text-center">
              <Zap className="h-12 w-12 text-muted-foreground mb-4 opacity-50" />
              <h3 className="text-lg font-semibold mb-2">No campaigns found</h3>
              <p className="text-sm text-muted-foreground mb-4">
                {searchQuery || selectedPlatform || selectedStatus
                  ? 'Try adjusting your filters'
                  : 'Get started by creating your first campaign'}
              </p>
              {!searchQuery && !selectedPlatform && !selectedStatus && (
                <Button>
                  <Plus className="h-4 w-4 mr-2" />
                  Create Campaign
                </Button>
              )}
            </CardContent>
          </Card>
        ) : (
          filteredCampaigns.map((campaign, index) => {
            const StatusIcon = statusStyles[campaign.status]?.icon || Clock
            return (
              <Card
                key={campaign.id}
                className="hover-lift animate-slide-up border-border/50 hover:border-primary/50 transition-all duration-300"
                style={{ animationDelay: `${index * 50}ms` }}
              >
                <CardContent className="py-5">
                  <div className="flex items-start gap-4">
                    {/* Left - Info */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-3 flex-wrap">
                        <h3 className="font-semibold text-base truncate">{campaign.name}</h3>
                        <span
                          className={`px-2.5 py-1 rounded-md text-xs font-medium capitalize whitespace-nowrap ${
                            platformStyles[campaign.platform]?.bg
                          } ${platformStyles[campaign.platform]?.text}`}
                        >
                          {campaign.platform}
                        </span>
                        <span
                          className={`px-2.5 py-1 rounded-md text-xs font-medium capitalize flex items-center gap-1.5 whitespace-nowrap ${
                            statusStyles[campaign.status]?.bg
                          } ${statusStyles[campaign.status]?.text}`}
                        >
                          <StatusIcon className="h-3 w-3" />
                          {campaign.status}
                        </span>
                      </div>

                      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm mb-4">
                        <div>
                          <p className="text-muted-foreground text-xs mb-0.5">Wallets</p>
                          <p className="font-medium">
                            {campaign.walletsCompleted}/
                            <span className="text-muted-foreground">
                              {campaign.walletsAssigned}
                            </span>
                          </p>
                        </div>
                        <div>
                          <p className="text-muted-foreground text-xs mb-0.5">Tasks</p>
                          <p className="font-medium">
                            {campaign.tasksCompleted}/
                            <span className="text-muted-foreground">
                              {campaign.tasksTotal}
                            </span>
                          </p>
                        </div>
                        <div>
                          <p className="text-muted-foreground text-xs mb-0.5">Progress</p>
                          <p className="font-medium">{campaign.progress}%</p>
                        </div>
                        <div>
                          <p className="text-muted-foreground text-xs mb-0.5">Reward</p>
                          <p className="font-medium text-primary">{campaign.estimatedReward}</p>
                        </div>
                      </div>

                      {/* Enhanced Progress bar */}
                      <div className="space-y-2">
                        <div className="flex items-center justify-between text-xs">
                          <span className="text-muted-foreground">Overall Progress</span>
                          <span className="font-medium">{campaign.progress}%</span>
                        </div>
                        <div className="relative h-2.5 bg-muted rounded-full overflow-hidden">
                          <div
                            className="h-full bg-gradient-to-r from-primary to-primary/80 rounded-full transition-all duration-500 ease-out relative overflow-hidden"
                            style={{ width: `${campaign.progress}%` }}
                          >
                            <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent animate-pulse-slow" />
                          </div>
                        </div>
                      </div>
                    </div>

                    {/* Right - Actions */}
                    <div className="flex items-center gap-2 flex-shrink-0">
                      <Button
                        variant="outline"
                        size="icon"
                        className="hover:bg-primary/10 hover:border-primary/50 transition-all"
                        title="View Campaign"
                      >
                        <ExternalLink className="h-4 w-4" />
                      </Button>
                      {campaign.status === 'running' ? (
                        <Button
                          variant="outline"
                          size="icon"
                          className="hover:bg-yellow-500/10 hover:border-yellow-500/50 hover:text-yellow-400 transition-all"
                          title="Pause Campaign"
                        >
                          <Pause className="h-4 w-4" />
                        </Button>
                      ) : campaign.status === 'paused' ? (
                        <Button
                          variant="outline"
                          size="icon"
                          className="hover:bg-green-500/10 hover:border-green-500/50 hover:text-green-400 transition-all"
                          title="Resume Campaign"
                        >
                          <Play className="h-4 w-4" />
                        </Button>
                      ) : null}
                      <Button
                        variant="ghost"
                        size="icon"
                        className="hover:bg-muted transition-all"
                        title="More Options"
                      >
                        <MoreVertical className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )
          })
        )}
      </div>
    </div>
  )
}
