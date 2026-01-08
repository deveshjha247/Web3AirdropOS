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
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Campaigns</h1>
          <p className="text-muted-foreground">
            Manage airdrop and reward campaigns
          </p>
        </div>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          Add Campaign
        </Button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Active</p>
                <p className="text-2xl font-bold">
                  {campaigns.filter((c) => c.status === 'running').length}
                </p>
              </div>
              <Play className="h-8 w-8 text-green-400" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Paused</p>
                <p className="text-2xl font-bold">
                  {campaigns.filter((c) => c.status === 'paused').length}
                </p>
              </div>
              <Pause className="h-8 w-8 text-yellow-400" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Completed</p>
                <p className="text-2xl font-bold">
                  {campaigns.filter((c) => c.status === 'completed').length}
                </p>
              </div>
              <CheckCircle className="h-8 w-8 text-blue-400" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Total Tasks</p>
                <p className="text-2xl font-bold">
                  {campaigns.reduce((sum, c) => sum + c.tasksCompleted, 0)}/
                  {campaigns.reduce((sum, c) => sum + c.tasksTotal, 0)}
                </p>
              </div>
              <Zap className="h-8 w-8 text-purple-400" />
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
        {filteredCampaigns.map((campaign) => {
          const StatusIcon = statusStyles[campaign.status]?.icon || Clock
          return (
            <Card key={campaign.id} className="hover:border-primary/50 transition-colors">
              <CardContent className="py-4">
                <div className="flex items-start gap-4">
                  {/* Left - Info */}
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-2">
                      <h3 className="font-medium">{campaign.name}</h3>
                      <span
                        className={`px-2 py-0.5 rounded text-xs capitalize ${
                          platformStyles[campaign.platform]?.bg
                        } ${platformStyles[campaign.platform]?.text}`}
                      >
                        {campaign.platform}
                      </span>
                      <span
                        className={`px-2 py-0.5 rounded text-xs capitalize flex items-center gap-1 ${
                          statusStyles[campaign.status]?.bg
                        } ${statusStyles[campaign.status]?.text}`}
                      >
                        <StatusIcon className="h-3 w-3" />
                        {campaign.status}
                      </span>
                    </div>

                    <div className="flex items-center gap-6 text-sm text-muted-foreground mb-3">
                      <span>
                        Wallets: {campaign.walletsCompleted}/
                        {campaign.walletsAssigned}
                      </span>
                      <span>
                        Tasks: {campaign.tasksCompleted}/{campaign.tasksTotal}
                      </span>
                      <span>Est. Reward: {campaign.estimatedReward}</span>
                    </div>

                    {/* Progress bar */}
                    <div className="flex items-center gap-3">
                      <div className="flex-1 h-2 bg-muted rounded-full overflow-hidden">
                        <div
                          className="h-full bg-primary transition-all"
                          style={{ width: `${campaign.progress}%` }}
                        />
                      </div>
                      <span className="text-sm font-medium w-12">
                        {campaign.progress}%
                      </span>
                    </div>
                  </div>

                  {/* Right - Actions */}
                  <div className="flex items-center gap-2">
                    <Button variant="outline" size="icon">
                      <ExternalLink className="h-4 w-4" />
                    </Button>
                    {campaign.status === 'running' ? (
                      <Button variant="outline" size="icon">
                        <Pause className="h-4 w-4" />
                      </Button>
                    ) : campaign.status === 'paused' ? (
                      <Button variant="outline" size="icon">
                        <Play className="h-4 w-4" />
                      </Button>
                    ) : null}
                    <Button variant="ghost" size="icon">
                      <MoreVertical className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          )
        })}
      </div>
    </div>
  )
}
