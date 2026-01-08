'use client'

import { DashboardStats } from '@/components/dashboard/stats'
import { RecentActivity } from '@/components/dashboard/recent-activity'
import { ActiveCampaigns } from '@/components/dashboard/active-campaigns'
import { QuickActions } from '@/components/dashboard/quick-actions'
import { Terminal } from '@/components/terminal/terminal'

export default function DashboardPage() {
  return (
    <div className="space-y-6 animate-fade-in">
      <div>
        <h1 className="text-3xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent">
          Dashboard
        </h1>
        <p className="text-muted-foreground mt-1">
          Web3AirdropOS Command Center
        </p>
      </div>

      {/* Stats Grid */}
      <div className="animate-slide-up" style={{ animationDelay: '0ms' }}>
        <DashboardStats />
      </div>

      {/* Main Content Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left Column - Activity & Campaigns */}
        <div className="lg:col-span-2 space-y-6">
          <div className="animate-slide-up" style={{ animationDelay: '100ms' }}>
            <ActiveCampaigns />
          </div>
          <div className="animate-slide-up" style={{ animationDelay: '200ms' }}>
            <RecentActivity />
          </div>
        </div>

        {/* Right Column - Quick Actions & Terminal */}
        <div className="space-y-6">
          <div className="animate-slide-up" style={{ animationDelay: '150ms' }}>
            <QuickActions />
          </div>
          <div className="animate-slide-up" style={{ animationDelay: '250ms' }}>
            <Terminal />
          </div>
        </div>
      </div>
    </div>
  )
}
