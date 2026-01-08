'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { CheckCircle, XCircle, Clock, AlertCircle } from 'lucide-react'

const activities = [
  {
    id: 1,
    action: 'Campaign task completed',
    details: 'Galxe - Follow Twitter',
    wallet: '0x1a2b...3c4d',
    status: 'success',
    time: '2 mins ago',
  },
  {
    id: 2,
    action: 'Content posted',
    details: 'Farcaster - Crypto thread',
    wallet: '0x5e6f...7g8h',
    status: 'success',
    time: '5 mins ago',
  },
  {
    id: 3,
    action: 'Task failed',
    details: 'Zealy - Quiz task timeout',
    wallet: '0x9i0j...1k2l',
    status: 'error',
    time: '12 mins ago',
  },
  {
    id: 4,
    action: 'Browser action pending',
    details: 'Manual verification required',
    wallet: '0x3m4n...5o6p',
    status: 'pending',
    time: '15 mins ago',
  },
  {
    id: 5,
    action: 'Wallet synced',
    details: 'Balance updated: 1.5 ETH',
    wallet: '0x7q8r...9s0t',
    status: 'success',
    time: '20 mins ago',
  },
]

const statusIcons = {
  success: <CheckCircle className="h-4 w-4 text-green-400" />,
  error: <XCircle className="h-4 w-4 text-red-400" />,
  pending: <Clock className="h-4 w-4 text-yellow-400" />,
  warning: <AlertCircle className="h-4 w-4 text-orange-400" />,
}

export function RecentActivity() {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">Recent Activity</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {activities.map((activity) => (
            <div
              key={activity.id}
              className="flex items-start gap-3 pb-4 border-b border-border last:border-0 last:pb-0"
            >
              <div className="mt-0.5">
                {statusIcons[activity.status as keyof typeof statusIcons]}
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium">{activity.action}</p>
                <p className="text-xs text-muted-foreground truncate">
                  {activity.details}
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  {activity.wallet}
                </p>
              </div>
              <span className="text-xs text-muted-foreground whitespace-nowrap">
                {activity.time}
              </span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
