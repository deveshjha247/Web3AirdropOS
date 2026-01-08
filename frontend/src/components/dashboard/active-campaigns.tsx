'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Play, Pause, ExternalLink } from 'lucide-react'

const campaigns = [
  {
    id: 1,
    name: 'Galxe - Layer3 Quest',
    platform: 'galxe',
    progress: 75,
    walletsActive: 12,
    walletsTotal: 16,
    tasksCompleted: 45,
    tasksTotal: 60,
    status: 'running',
  },
  {
    id: 2,
    name: 'Zealy - Community Sprint',
    platform: 'zealy',
    progress: 40,
    walletsActive: 8,
    walletsTotal: 10,
    tasksCompleted: 16,
    tasksTotal: 40,
    status: 'running',
  },
  {
    id: 3,
    name: 'Layer3 - DeFi Discovery',
    platform: 'layer3',
    progress: 90,
    walletsActive: 5,
    walletsTotal: 5,
    tasksCompleted: 18,
    tasksTotal: 20,
    status: 'paused',
  },
]

const platformColors: Record<string, string> = {
  galxe: 'bg-purple-500/20 text-purple-400',
  zealy: 'bg-blue-500/20 text-blue-400',
  layer3: 'bg-green-500/20 text-green-400',
}

export function ActiveCampaigns() {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="text-lg">Active Campaigns</CardTitle>
        <Button variant="outline" size="sm">
          View All
        </Button>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {campaigns.map((campaign) => (
            <div
              key={campaign.id}
              className="p-4 rounded-lg border border-border bg-muted/50"
            >
              <div className="flex items-start justify-between mb-3">
                <div>
                  <div className="flex items-center gap-2">
                    <h3 className="font-medium">{campaign.name}</h3>
                    <span
                      className={`px-2 py-0.5 rounded text-xs ${
                        platformColors[campaign.platform]
                      }`}
                    >
                      {campaign.platform}
                    </span>
                  </div>
                  <p className="text-xs text-muted-foreground mt-1">
                    {campaign.walletsActive}/{campaign.walletsTotal} wallets •{' '}
                    {campaign.tasksCompleted}/{campaign.tasksTotal} tasks
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <Button variant="ghost" size="icon" className="h-8 w-8">
                    {campaign.status === 'running' ? (
                      <Pause className="h-4 w-4" />
                    ) : (
                      <Play className="h-4 w-4" />
                    )}
                  </Button>
                  <Button variant="ghost" size="icon" className="h-8 w-8">
                    <ExternalLink className="h-4 w-4" />
                  </Button>
                </div>
              </div>

              {/* Progress bar */}
              <div className="h-2 bg-muted rounded-full overflow-hidden">
                <div
                  className="h-full bg-primary transition-all"
                  style={{ width: `${campaign.progress}%` }}
                />
              </div>
              <div className="flex justify-between mt-1">
                <span className="text-xs text-muted-foreground">Progress</span>
                <span className="text-xs font-medium">{campaign.progress}%</span>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
