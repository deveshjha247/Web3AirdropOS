'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { cn } from '@/lib/utils'
import {
  LayoutDashboard,
  Wallet,
  Users,
  Globe,
  Zap,
  FileText,
  Bot,
  Settings,
  Shield,
  Monitor,
} from 'lucide-react'

const navigation = [
  { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
  { name: 'Wallets', href: '/wallets', icon: Wallet },
  { name: 'Accounts', href: '/accounts', icon: Users },
  { name: 'Campaigns', href: '/campaigns', icon: Zap },
  { name: 'Browser', href: '/browser', icon: Monitor },
  { name: 'Content', href: '/content', icon: FileText },
  { name: 'Automation', href: '/automation', icon: Bot },
  { name: 'Proxies', href: '/proxies', icon: Shield },
  { name: 'Settings', href: '/settings', icon: Settings },
]

export function Sidebar() {
  const pathname = usePathname()

  return (
    <aside className="w-64 bg-card/50 backdrop-blur-sm border-r border-border flex flex-col">
      {/* Logo */}
      <div className="h-16 flex items-center px-6 border-b border-border">
        <Link
          href="/"
          className="flex items-center group hover:opacity-80 transition-opacity"
        >
          <div className="h-10 w-10 rounded-lg bg-gradient-to-br from-primary to-primary/60 flex items-center justify-center mr-3 group-hover:scale-105 transition-transform">
            <Globe className="h-6 w-6 text-primary-foreground" />
          </div>
          <span className="text-xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent">
            Web3AirdropOS
          </span>
        </Link>
      </div>

      {/* Navigation */}
      <nav className="flex-1 p-4 space-y-1 overflow-y-auto">
        {navigation.map((item, index) => {
          const isActive = pathname === item.href
          return (
            <Link
              key={item.name}
              href={item.href}
              className={cn(
                'flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-200 relative group',
                isActive
                  ? 'bg-primary text-primary-foreground shadow-lg shadow-primary/20'
                  : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
              )}
              style={{ animationDelay: `${index * 30}ms` }}
            >
              {isActive && (
                <div className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-primary-foreground rounded-r-full" />
              )}
              <item.icon
                className={cn(
                  'h-5 w-5 transition-transform',
                  isActive ? 'scale-110' : 'group-hover:scale-105'
                )}
              />
              <span className="flex-1">{item.name}</span>
              {isActive && (
                <div className="h-1.5 w-1.5 rounded-full bg-primary-foreground animate-pulse" />
              )}
            </Link>
          )
        })}
      </nav>

      {/* Status */}
      <div className="p-4 border-t border-border">
        <div className="flex items-center gap-2.5 text-sm p-2 rounded-lg bg-muted/30">
          <div className="relative">
            <div className="h-2.5 w-2.5 rounded-full bg-green-500" />
            <div className="absolute inset-0 h-2.5 w-2.5 rounded-full bg-green-500 animate-ping opacity-75" />
          </div>
          <span className="text-muted-foreground font-medium">System Online</span>
        </div>
      </div>
    </aside>
  )
}
