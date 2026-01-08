import type { Metadata } from 'next'
import { Providers } from '@/components/providers'
import { Sidebar } from '@/components/layout/sidebar'
import { Header } from '@/components/layout/header'
import { Toaster } from '@/components/ui/toaster'

export const metadata: Metadata = {
  title: 'Dashboard | Web3AirdropOS',
  description: 'Web3AirdropOS Command Center',
}

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <Providers>
      <div className="flex h-screen overflow-hidden">
        <Sidebar />
        <div className="flex-1 flex flex-col overflow-hidden">
          <Header />
          <main className="flex-1 overflow-auto p-6">
            {children}
          </main>
        </div>
      </div>
      <Toaster />
    </Providers>
  )
}
