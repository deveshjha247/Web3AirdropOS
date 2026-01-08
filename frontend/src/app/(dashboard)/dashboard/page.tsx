'use client'

import { DashboardStats } from '@/components/dashboard/stats'
import { RecentActivity } from '@/components/dashboard/recent-activity'
import { ActiveCampaigns } from '@/components/dashboard/active-campaigns'
import { QuickActions } from '@/components/dashboard/quick-actions'
import { Terminal } from '@/components/terminal/terminal'

export default function DashboardPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Dashboard</h1>
        <p className="text-muted-foreground">Web3AirdropOS Command Center</p>
      </div>

      {/* Stats Grid */}
      <DashboardStats />

      {/* Main Content Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left Column - Activity & Campaigns */}
        <div className="lg:col-span-2 space-y-6">
          <ActiveCampaigns />
          <RecentActivity />
        </div>

        {/* Right Column - Quick Actions & Terminal */}
        <div className="space-y-6">
          <QuickActions />
          <Terminal />
        </div>
      </div>
    </div>
  )
}
