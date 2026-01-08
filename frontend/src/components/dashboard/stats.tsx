'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Wallet, Users, Zap, Activity } from 'lucide-react'

const stats = [
  {
    name: 'Total Wallets',
    value: '24',
    change: '+3 this week',
    icon: Wallet,
    color: 'text-blue-400',
  },
  {
    name: 'Active Accounts',
    value: '12',
    change: '4 platforms',
    icon: Users,
    color: 'text-green-400',
  },
  {
    name: 'Active Campaigns',
    value: '8',
    change: '156 tasks completed',
    icon: Zap,
    color: 'text-purple-400',
  },
  {
    name: 'Jobs Running',
    value: '3',
    change: '2 scheduled',
    icon: Activity,
    color: 'text-orange-400',
  },
]

export function DashboardStats() {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      {stats.map((stat) => (
        <Card key={stat.name}>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {stat.name}
            </CardTitle>
            <stat.icon className={`h-5 w-5 ${stat.color}`} />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{stat.value}</div>
            <p className="text-xs text-muted-foreground mt-1">{stat.change}</p>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
