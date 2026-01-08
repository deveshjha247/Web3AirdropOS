'use client'

import { useEffect, useState } from 'react'

type ToastProps = {
  id: string
  title?: string
  description?: string
  variant?: 'default' | 'destructive'
}

type ToastState = {
  toasts: ToastProps[]
  addToast: (toast: Omit<ToastProps, 'id'>) => void
  removeToast: (id: string) => void
}

// Simple toast implementation
export function Toaster() {
  const [toasts, setToasts] = useState<ToastProps[]>([])

  return (
    <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2">
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className={`px-4 py-3 rounded-lg shadow-lg ${
            toast.variant === 'destructive'
              ? 'bg-destructive text-destructive-foreground'
              : 'bg-card border border-border'
          }`}
        >
          {toast.title && <p className="font-medium">{toast.title}</p>}
          {toast.description && (
            <p className="text-sm text-muted-foreground">{toast.description}</p>
          )}
        </div>
      ))}
    </div>
  )
}

export function toast(props: Omit<ToastProps, 'id'>) {
  // This would typically use a global state manager
  console.log('Toast:', props)
}
