'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import {
  Wallet,
  UserPlus,
  Zap,
  Bot,
  FileText,
  RefreshCw,
} from 'lucide-react'

const quickActions = [
  { name: 'Add Wallet', icon: Wallet, color: 'text-blue-400' },
  { name: 'Link Account', icon: UserPlus, color: 'text-green-400' },
  { name: 'New Campaign', icon: Zap, color: 'text-purple-400' },
  { name: 'Generate Content', icon: FileText, color: 'text-orange-400' },
  { name: 'Start Automation', icon: Bot, color: 'text-pink-400' },
  { name: 'Sync All', icon: RefreshCw, color: 'text-cyan-400' },
]

export function QuickActions() {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">Quick Actions</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 gap-2">
          {quickActions.map((action) => (
            <Button
              key={action.name}
              variant="outline"
              className="h-auto py-3 flex flex-col items-center gap-2"
            >
              <action.icon className={`h-5 w-5 ${action.color}`} />
              <span className="text-xs">{action.name}</span>
            </Button>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
