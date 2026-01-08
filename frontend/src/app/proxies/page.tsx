'use client'

import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import {
  Plus,
  Search,
  MoreVertical,
  Shield,
  CheckCircle,
  XCircle,
  RefreshCw,
  Trash2,
  Upload,
} from 'lucide-react'

const proxies = [
  {
    id: '1',
    name: 'US Proxy 1',
    type: 'http',
    host: '192.168.1.100',
    port: 8080,
    country: 'US',
    status: 'active',
    latency: 45,
    lastTested: '5 mins ago',
    assignedTo: 3,
  },
  {
    id: '2',
    name: 'EU Proxy 2',
    type: 'socks5',
    host: '10.0.0.50',
    port: 1080,
    country: 'DE',
    status: 'active',
    latency: 78,
    lastTested: '10 mins ago',
    assignedTo: 2,
  },
  {
    id: '3',
    name: 'Asia Proxy 1',
    type: 'http',
    host: '172.16.0.25',
    port: 3128,
    country: 'SG',
    status: 'inactive',
    latency: null,
    lastTested: '1 hour ago',
    assignedTo: 0,
  },
  {
    id: '4',
    name: 'US Proxy 2',
    type: 'socks5',
    host: '192.168.2.100',
    port: 1080,
    country: 'US',
    status: 'active',
    latency: 32,
    lastTested: '2 mins ago',
    assignedTo: 5,
  },
]

const countryFlags: Record<string, string> = {
  US: '🇺🇸',
  DE: '🇩🇪',
  SG: '🇸🇬',
  GB: '🇬🇧',
  JP: '🇯🇵',
}

export default function ProxiesPage() {
  const [searchQuery, setSearchQuery] = useState('')

  const filteredProxies = proxies.filter(
    (proxy) =>
      proxy.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      proxy.host.includes(searchQuery)
  )

  const activeCount = proxies.filter((p) => p.status === 'active').length
  const avgLatency = Math.round(
    proxies
      .filter((p) => p.latency)
      .reduce((sum, p) => sum + (p.latency || 0), 0) /
      proxies.filter((p) => p.latency).length
  )

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Proxies</h1>
          <p className="text-muted-foreground">
            Manage proxy configurations for isolation
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline">
            <Upload className="h-4 w-4 mr-2" />
            Bulk Import
          </Button>
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            Add Proxy
          </Button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Total Proxies</p>
                <p className="text-2xl font-bold">{proxies.length}</p>
              </div>
              <Shield className="h-8 w-8 text-blue-400" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Active</p>
                <p className="text-2xl font-bold">{activeCount}</p>
              </div>
              <CheckCircle className="h-8 w-8 text-green-400" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Avg Latency</p>
                <p className="text-2xl font-bold">{avgLatency}ms</p>
              </div>
              <RefreshCw className="h-8 w-8 text-purple-400" />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Assigned</p>
                <p className="text-2xl font-bold">
                  {proxies.reduce((sum, p) => sum + p.assignedTo, 0)}
                </p>
              </div>
              <Shield className="h-8 w-8 text-orange-400" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Search & Actions */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search proxies..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full h-10 pl-10 pr-4 rounded-lg bg-muted border border-input text-sm focus:outline-none focus:ring-2 focus:ring-primary"
          />
        </div>
        <Button variant="outline" size="sm">
          <RefreshCw className="h-4 w-4 mr-2" />
          Test All
        </Button>
      </div>

      {/* Proxies Table */}
      <Card>
        <CardContent className="p-0">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border">
                <th className="text-left p-4 text-sm font-medium text-muted-foreground">
                  Name
                </th>
                <th className="text-left p-4 text-sm font-medium text-muted-foreground">
                  Type
                </th>
                <th className="text-left p-4 text-sm font-medium text-muted-foreground">
                  Host
                </th>
                <th className="text-left p-4 text-sm font-medium text-muted-foreground">
                  Country
                </th>
                <th className="text-left p-4 text-sm font-medium text-muted-foreground">
                  Status
                </th>
                <th className="text-left p-4 text-sm font-medium text-muted-foreground">
                  Latency
                </th>
                <th className="text-left p-4 text-sm font-medium text-muted-foreground">
                  Assigned
                </th>
                <th className="text-right p-4 text-sm font-medium text-muted-foreground">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody>
              {filteredProxies.map((proxy) => (
                <tr
                  key={proxy.id}
                  className="border-b border-border hover:bg-muted/50"
                >
                  <td className="p-4">
                    <span className="font-medium">{proxy.name}</span>
                  </td>
                  <td className="p-4">
                    <span className="px-2 py-0.5 rounded text-xs bg-muted uppercase">
                      {proxy.type}
                    </span>
                  </td>
                  <td className="p-4 font-mono text-sm">
                    {proxy.host}:{proxy.port}
                  </td>
                  <td className="p-4">
                    <span className="flex items-center gap-1">
                      {countryFlags[proxy.country]} {proxy.country}
                    </span>
                  </td>
                  <td className="p-4">
                    <span
                      className={`flex items-center gap-1 ${
                        proxy.status === 'active'
                          ? 'text-green-400'
                          : 'text-red-400'
                      }`}
                    >
                      {proxy.status === 'active' ? (
                        <CheckCircle className="h-4 w-4" />
                      ) : (
                        <XCircle className="h-4 w-4" />
                      )}
                      {proxy.status}
                    </span>
                  </td>
                  <td className="p-4">
                    {proxy.latency ? (
                      <span
                        className={
                          proxy.latency < 50
                            ? 'text-green-400'
                            : proxy.latency < 100
                            ? 'text-yellow-400'
                            : 'text-red-400'
                        }
                      >
                        {proxy.latency}ms
                      </span>
                    ) : (
                      <span className="text-muted-foreground">-</span>
                    )}
                  </td>
                  <td className="p-4">
                    {proxy.assignedTo} profiles
                  </td>
                  <td className="p-4 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <Button variant="ghost" size="icon" className="h-8 w-8">
                        <RefreshCw className="h-4 w-4" />
                      </Button>
                      <Button variant="ghost" size="icon" className="h-8 w-8">
                        <Trash2 className="h-4 w-4" />
                      </Button>
                      <Button variant="ghost" size="icon" className="h-8 w-8">
                        <MoreVertical className="h-4 w-4" />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </CardContent>
      </Card>
    </div>
  )
}
