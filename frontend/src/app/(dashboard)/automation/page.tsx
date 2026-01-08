'use client'

import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import {
  Plus,
  Search,
  Play,
  Pause,
  Clock,
  MoreVertical,
  Bot,
  Calendar,
  Activity,
  CheckCircle,
  XCircle,
} from 'lucide-react'

const jobs = [
  {
    id: '1',
    name: 'Daily Farcaster Engagement',
    type: 'scheduled_post',
    schedule: '0 9,14,20 * * *',
    status: 'active',
    lastRun: '2 hours ago',
    nextRun: 'in 4 hours',
    runsCompleted: 45,
    runsFailed: 2,
    accounts: ['cryptobuilder.eth', 'web3enthusiast'],
  },
  {
    id: '2',
    name: 'Campaign Task Auto-Execute',
    type: 'campaign_execute',
    schedule: 'Every 30 minutes',
    status: 'active',
    lastRun: '15 mins ago',
    nextRun: 'in 15 mins',
    runsCompleted: 120,
    runsFailed: 5,
    campaigns: ['Galxe Quest', 'Zealy Sprint'],
  },
  {
    id: '3',
    name: 'Wallet Balance Monitor',
    type: 'wallet_sync',
    schedule: 'Every hour',
    status: 'paused',
    lastRun: '3 hours ago',
    nextRun: 'Paused',
    runsCompleted: 72,
    runsFailed: 0,
    wallets: 24,
  },
  {
    id: '4',
    name: 'Twitter Reply Bot',
    type: 'social_engage',
    schedule: 'Every 2 hours',
    status: 'active',
    lastRun: '1 hour ago',
    nextRun: 'in 1 hour',
    runsCompleted: 30,
    runsFailed: 1,
    accounts: ['@web3enthusiast'],
  },
]

const jobTypes = [
  { value: 'scheduled_post', label: 'Scheduled Post' },
  { value: 'campaign_execute', label: 'Campaign Execute' },
  { value: 'wallet_sync', label: 'Wallet Sync' },
  { value: 'social_engage', label: 'Social Engage' },
  { value: 'content_generate', label: 'Content Generate' },
]

const jobTypeStyles: Record<string, { bg: string; text: string }> = {
  scheduled_post: { bg: 'bg-blue-500/20', text: 'text-blue-400' },
  campaign_execute: { bg: 'bg-purple-500/20', text: 'text-purple-400' },
  wallet_sync: { bg: 'bg-green-500/20', text: 'text-green-400' },
  social_engage: { bg: 'bg-pink-500/20', text: 'text-pink-400' },
  content_generate: { bg: 'bg-orange-500/20', text: 'text-orange-400' },
}

export default function AutomationPage() {
  const [searchQuery, setSearchQuery] = useState('')

  const filteredJobs = jobs.filter((job) =>
    job.name.toLowerCase().includes(searchQuery.toLowerCase())
  )

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Automation</h1>
          <p className="text-muted-foreground">
            Scheduled jobs and automated workflows
          </p>
        </div>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          Create Job
        </Button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Active Jobs</p>
                <p className="text-2xl font-bold">
                  {jobs.filter((j) => j.status === 'active').length}
                </p>
              </div>
              <Activity className="h-8 w-8 text-green-400" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Total Runs Today</p>
                <p className="text-2xl font-bold">156</p>
              </div>
              <Bot className="h-8 w-8 text-blue-400" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Success Rate</p>
                <p className="text-2xl font-bold">97.2%</p>
              </div>
              <CheckCircle className="h-8 w-8 text-purple-400" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Next Scheduled</p>
                <p className="text-2xl font-bold">15m</p>
              </div>
              <Clock className="h-8 w-8 text-orange-400" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Search */}
      <div className="relative max-w-sm">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <input
          type="text"
          placeholder="Search jobs..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="w-full h-10 pl-10 pr-4 rounded-lg bg-muted border border-input text-sm focus:outline-none focus:ring-2 focus:ring-primary"
        />
      </div>

      {/* Jobs List */}
      <div className="space-y-4">
        {filteredJobs.map((job) => (
          <Card key={job.id} className="hover:border-primary/50 transition-colors">
            <CardContent className="py-4">
              <div className="flex items-start gap-4">
                {/* Left - Info */}
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-2">
                    <h3 className="font-medium">{job.name}</h3>
                    <span
                      className={`px-2 py-0.5 rounded text-xs ${
                        jobTypeStyles[job.type]?.bg
                      } ${jobTypeStyles[job.type]?.text}`}
                    >
                      {jobTypes.find((t) => t.value === job.type)?.label}
                    </span>
                    <span
                      className={`px-2 py-0.5 rounded text-xs ${
                        job.status === 'active'
                          ? 'bg-green-500/20 text-green-400'
                          : 'bg-yellow-500/20 text-yellow-400'
                      }`}
                    >
                      {job.status}
                    </span>
                  </div>

                  <div className="flex items-center gap-6 text-sm text-muted-foreground mb-2">
                    <span className="flex items-center gap-1">
                      <Calendar className="h-3 w-3" />
                      {job.schedule}
                    </span>
                    <span className="flex items-center gap-1">
                      <CheckCircle className="h-3 w-3 text-green-400" />
                      {job.runsCompleted} completed
                    </span>
                    <span className="flex items-center gap-1">
                      <XCircle className="h-3 w-3 text-red-400" />
                      {job.runsFailed} failed
                    </span>
                  </div>

                  <div className="flex items-center gap-4 text-xs text-muted-foreground">
                    <span>Last run: {job.lastRun}</span>
                    <span>Next run: {job.nextRun}</span>
                  </div>
                </div>

                {/* Right - Actions */}
                <div className="flex items-center gap-2">
                  {job.status === 'active' ? (
                    <Button variant="outline" size="sm">
                      <Pause className="h-4 w-4 mr-1" />
                      Pause
                    </Button>
                  ) : (
                    <Button variant="outline" size="sm">
                      <Play className="h-4 w-4 mr-1" />
                      Resume
                    </Button>
                  )}
                  <Button variant="outline" size="sm">
                    Run Now
                  </Button>
                  <Button variant="ghost" size="icon">
                    <MoreVertical className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
