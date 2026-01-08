'use client'

import { useEffect } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'

export default function RegisterPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const plan = searchParams.get('plan')
  
  useEffect(() => {
    // Redirect to login page with register mode
    const url = plan ? `/login?mode=register&plan=${plan}` : '/login?mode=register'
    router.replace(url)
  }, [router, plan])

  return (
    <div className="min-h-screen bg-zinc-950 flex items-center justify-center">
      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-violet-500"></div>
    </div>
  )
}
