export default function LoginLayout({
  children,
}: {
  children: React.ReactNode
}) {
  // Login page has its own layout without sidebar
  return <>{children}</>
}
