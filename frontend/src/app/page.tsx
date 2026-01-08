'use client';

import { useState } from 'react';
import Link from 'next/link';
import { 
  Wallet, 
  Target, 
  Monitor, 
  Terminal, 
  Sparkles, 
  Bell, 
  Shield,
  ChevronDown,
  Check,
  ArrowRight,
  Zap,
  Users,
  Building2,
  Menu,
  X
} from 'lucide-react';

// =====================================================
// NAVBAR
// =====================================================
function Navbar() {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  return (
    <nav className="fixed top-0 left-0 right-0 z-50 bg-zinc-950/80 backdrop-blur-xl border-b border-zinc-800">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          {/* Logo */}
          <Link href="/" className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-violet-500 to-fuchsia-500 flex items-center justify-center">
              <Zap className="w-5 h-5 text-white" />
            </div>
            <span className="text-lg font-bold text-white">Web3AirdropOS</span>
          </Link>

          {/* Desktop Nav */}
          <div className="hidden md:flex items-center gap-8">
            <a href="#features" className="text-sm text-zinc-400 hover:text-white transition-colors">Features</a>
            <a href="#how-it-works" className="text-sm text-zinc-400 hover:text-white transition-colors">How it works</a>
            <a href="#pricing" className="text-sm text-zinc-400 hover:text-white transition-colors">Pricing</a>
            <a href="/docs" className="text-sm text-zinc-400 hover:text-white transition-colors">Docs</a>
          </div>

          {/* Auth Buttons */}
          <div className="hidden md:flex items-center gap-3">
            <Link 
              href="/login" 
              className="px-4 py-2 text-sm font-medium text-zinc-300 hover:text-white transition-colors"
            >
              Sign in
            </Link>
            <Link 
              href="/register" 
              className="px-4 py-2 text-sm font-medium text-white bg-violet-600 hover:bg-violet-500 rounded-lg transition-colors"
            >
              Create account
            </Link>
          </div>

          {/* Mobile Menu Button */}
          <button 
            className="md:hidden p-2 text-zinc-400 hover:text-white"
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
          >
            {mobileMenuOpen ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
          </button>
        </div>

        {/* Mobile Menu */}
        {mobileMenuOpen && (
          <div className="md:hidden py-4 border-t border-zinc-800">
            <div className="flex flex-col gap-4">
              <a href="#features" className="text-sm text-zinc-400 hover:text-white">Features</a>
              <a href="#how-it-works" className="text-sm text-zinc-400 hover:text-white">How it works</a>
              <a href="#pricing" className="text-sm text-zinc-400 hover:text-white">Pricing</a>
              <a href="/docs" className="text-sm text-zinc-400 hover:text-white">Docs</a>
              <div className="flex flex-col gap-2 pt-4 border-t border-zinc-800">
                <Link href="/login" className="px-4 py-2 text-sm font-medium text-center text-zinc-300 border border-zinc-700 rounded-lg">
                  Sign in
                </Link>
                <Link href="/register" className="px-4 py-2 text-sm font-medium text-center text-white bg-violet-600 rounded-lg">
                  Create account
                </Link>
              </div>
            </div>
          </div>
        )}
      </div>
    </nav>
  );
}

// =====================================================
// HERO SECTION
// =====================================================
function HeroSection() {
  return (
    <section className="relative pt-32 pb-20 px-4 overflow-hidden">
      {/* Background gradient */}
      <div className="absolute inset-0 bg-gradient-to-b from-violet-950/20 via-transparent to-transparent" />
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[600px] bg-violet-500/10 blur-[120px] rounded-full" />
      
      <div className="relative max-w-5xl mx-auto text-center">
        {/* Badge */}
        <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-zinc-800/50 border border-zinc-700 mb-8">
          <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
          <span className="text-sm text-zinc-300">Now with Farcaster & Telegram integrations</span>
        </div>

        {/* Headline */}
        <h1 className="text-4xl sm:text-5xl md:text-6xl lg:text-7xl font-bold text-white mb-6 leading-tight">
          All-in-one crypto
          <br />
          <span className="bg-gradient-to-r from-violet-400 via-fuchsia-400 to-violet-400 bg-clip-text text-transparent">
            operations platform
          </span>
        </h1>

        {/* Subheadline */}
        <p className="text-lg sm:text-xl text-zinc-400 max-w-2xl mx-auto mb-10">
          Manage multi-wallet operations, automate airdrop campaigns, and coordinate social actions across platforms — all from a single dashboard.
        </p>

        {/* CTA Buttons */}
        <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
          <Link 
            href="/register" 
            className="w-full sm:w-auto px-8 py-4 text-base font-semibold text-white bg-violet-600 hover:bg-violet-500 rounded-xl transition-all hover:shadow-lg hover:shadow-violet-500/25 flex items-center justify-center gap-2"
          >
            Create free account
            <ArrowRight className="w-5 h-5" />
          </Link>
          <Link 
            href="/login" 
            className="w-full sm:w-auto px-8 py-4 text-base font-semibold text-zinc-300 bg-zinc-800 hover:bg-zinc-700 border border-zinc-700 rounded-xl transition-colors flex items-center justify-center"
          >
            Sign in to dashboard
          </Link>
        </div>

        {/* Trust indicators */}
        <div className="mt-16 flex flex-wrap items-center justify-center gap-8 text-zinc-500 text-sm">
          <div className="flex items-center gap-2">
            <Shield className="w-4 h-4" />
            <span>End-to-end encrypted</span>
          </div>
          <div className="flex items-center gap-2">
            <Wallet className="w-4 h-4" />
            <span>EVM + Solana support</span>
          </div>
          <div className="flex items-center gap-2">
            <Users className="w-4 h-4" />
            <span>Self-hosted option</span>
          </div>
        </div>
      </div>
    </section>
  );
}

// =====================================================
// FEATURES GRID
// =====================================================
const features = [
  {
    icon: Wallet,
    title: 'Multi-Wallet Management',
    description: 'Manage EVM and Solana wallets with groups, tags, and campaign assignments. Track balances across all chains.',
    color: 'from-blue-500 to-cyan-500'
  },
  {
    icon: Target,
    title: 'Campaign Task Automation',
    description: 'Execute airdrop tasks across Galxe, Zealy, Layer3, and more. Bulk operations with dependency handling.',
    color: 'from-violet-500 to-fuchsia-500'
  },
  {
    icon: Monitor,
    title: 'Embedded Browser Workspace',
    description: 'Full Chromium browser in dashboard with session isolation, proxy support, and manual takeover when needed.',
    color: 'from-emerald-500 to-teal-500'
  },
  {
    icon: Terminal,
    title: 'Real-time Terminal',
    description: 'Live WebSocket-powered logs showing every operation as it happens. Color-coded status and action visibility.',
    color: 'from-orange-500 to-amber-500'
  },
  {
    icon: Sparkles,
    title: 'AI Content Drafts',
    description: 'Generate platform-optimized content with approve-first workflow. Review and publish when ready.',
    color: 'from-pink-500 to-rose-500'
  },
  {
    icon: Bell,
    title: 'Smart Notifications',
    description: 'Get alerts for completed tasks, failed operations, and approval requests. Never miss important updates.',
    color: 'from-indigo-500 to-violet-500'
  },
  {
    icon: Shield,
    title: 'Proof Tracking',
    description: 'Automatic proof capture with post URLs, transaction hashes, and screenshots. Complete audit trail.',
    color: 'from-cyan-500 to-blue-500'
  }
];

function FeaturesSection() {
  return (
    <section id="features" className="py-20 px-4">
      <div className="max-w-7xl mx-auto">
        {/* Section header */}
        <div className="text-center mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold text-white mb-4">
            Everything you need to operate at scale
          </h2>
          <p className="text-lg text-zinc-400 max-w-2xl mx-auto">
            Purpose-built tools for managing multi-wallet campaigns and social operations across Web3 platforms.
          </p>
        </div>

        {/* Features grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {features.map((feature, index) => (
            <div 
              key={index}
              className="group p-6 rounded-2xl bg-zinc-900/50 border border-zinc-800 hover:border-zinc-700 transition-all hover:bg-zinc-900"
            >
              <div className={`w-12 h-12 rounded-xl bg-gradient-to-br ${feature.color} flex items-center justify-center mb-4`}>
                <feature.icon className="w-6 h-6 text-white" />
              </div>
              <h3 className="text-lg font-semibold text-white mb-2">{feature.title}</h3>
              <p className="text-zinc-400 text-sm leading-relaxed">{feature.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

// =====================================================
// HOW IT WORKS
// =====================================================
const steps = [
  {
    step: '01',
    title: 'Connect your wallets & accounts',
    description: 'Import your EVM and Solana wallets, connect platform accounts (Farcaster, Telegram, Twitter), and configure proxies for isolation.'
  },
  {
    step: '02',
    title: 'Set up campaigns & tasks',
    description: 'Create campaigns for airdrops, assign wallets and accounts, configure task sequences with dependencies and schedules.'
  },
  {
    step: '03',
    title: 'Execute & monitor in real-time',
    description: 'Run automated tasks, review AI-generated content before publishing, and track progress with complete proof capture.'
  }
];

function HowItWorksSection() {
  return (
    <section id="how-it-works" className="py-20 px-4 bg-zinc-900/30">
      <div className="max-w-5xl mx-auto">
        {/* Section header */}
        <div className="text-center mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold text-white mb-4">
            How it works
          </h2>
          <p className="text-lg text-zinc-400">
            Get started in minutes with a simple three-step process
          </p>
        </div>

        {/* Steps */}
        <div className="space-y-8">
          {steps.map((item, index) => (
            <div 
              key={index}
              className="flex gap-6 p-6 rounded-2xl bg-zinc-900/50 border border-zinc-800"
            >
              <div className="flex-shrink-0 w-16 h-16 rounded-xl bg-gradient-to-br from-violet-600 to-fuchsia-600 flex items-center justify-center">
                <span className="text-xl font-bold text-white">{item.step}</span>
              </div>
              <div>
                <h3 className="text-xl font-semibold text-white mb-2">{item.title}</h3>
                <p className="text-zinc-400">{item.description}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

// =====================================================
// PRICING
// =====================================================
const plans = [
  {
    name: 'Free',
    price: '$0',
    period: 'forever',
    description: 'Perfect for getting started',
    icon: Zap,
    features: [
      'Up to 5 wallets',
      '2 platform accounts',
      '1 active campaign',
      'Basic terminal logs',
      '10 AI content generations/month',
      'Community support'
    ],
    cta: 'Get started free',
    href: '/register',
    highlighted: false
  },
  {
    name: 'Pro',
    price: '$29',
    period: '/month',
    description: 'For serious operators',
    icon: Users,
    features: [
      'Unlimited wallets',
      'Unlimited accounts',
      'Unlimited campaigns',
      'Full audit logging',
      '500 AI generations/month',
      'Browser workspace',
      'Priority support',
      'API access'
    ],
    cta: 'Start 14-day trial',
    href: '/register?plan=pro',
    highlighted: true
  },
  {
    name: 'Team',
    price: '$99',
    period: '/month',
    description: 'For teams & agencies',
    icon: Building2,
    features: [
      'Everything in Pro',
      'Up to 10 team members',
      'Role-based permissions',
      'Shared wallet groups',
      'Team audit logs',
      'Dedicated support',
      'Custom integrations',
      'SLA guarantee'
    ],
    cta: 'Contact sales',
    href: '/contact',
    highlighted: false
  }
];

function PricingSection() {
  return (
    <section id="pricing" className="py-20 px-4">
      <div className="max-w-6xl mx-auto">
        {/* Section header */}
        <div className="text-center mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold text-white mb-4">
            Simple, transparent pricing
          </h2>
          <p className="text-lg text-zinc-400">
            Start free, upgrade when you need more power
          </p>
        </div>

        {/* Pricing cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {plans.map((plan, index) => (
            <div 
              key={index}
              className={`relative p-6 rounded-2xl border transition-all ${
                plan.highlighted 
                  ? 'bg-gradient-to-b from-violet-950/50 to-zinc-900 border-violet-500/50 shadow-lg shadow-violet-500/10' 
                  : 'bg-zinc-900/50 border-zinc-800 hover:border-zinc-700'
              }`}
            >
              {plan.highlighted && (
                <div className="absolute -top-3 left-1/2 -translate-x-1/2 px-3 py-1 rounded-full bg-violet-600 text-xs font-medium text-white">
                  Most popular
                </div>
              )}
              
              <div className="mb-6">
                <div className={`w-10 h-10 rounded-lg ${plan.highlighted ? 'bg-violet-600' : 'bg-zinc-800'} flex items-center justify-center mb-4`}>
                  <plan.icon className="w-5 h-5 text-white" />
                </div>
                <h3 className="text-xl font-semibold text-white">{plan.name}</h3>
                <p className="text-sm text-zinc-400 mt-1">{plan.description}</p>
              </div>

              <div className="mb-6">
                <span className="text-4xl font-bold text-white">{plan.price}</span>
                <span className="text-zinc-400">{plan.period}</span>
              </div>

              <ul className="space-y-3 mb-8">
                {plan.features.map((feature, i) => (
                  <li key={i} className="flex items-start gap-3">
                    <Check className="w-5 h-5 text-emerald-500 flex-shrink-0 mt-0.5" />
                    <span className="text-sm text-zinc-300">{feature}</span>
                  </li>
                ))}
              </ul>

              <Link 
                href={plan.href}
                className={`block w-full py-3 text-center font-medium rounded-lg transition-colors ${
                  plan.highlighted 
                    ? 'bg-violet-600 hover:bg-violet-500 text-white' 
                    : 'bg-zinc-800 hover:bg-zinc-700 text-zinc-300 border border-zinc-700'
                }`}
              >
                {plan.cta}
              </Link>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

// =====================================================
// FAQ
// =====================================================
const faqs = [
  {
    question: 'How are my private keys stored?',
    answer: 'All private keys are encrypted using AES-256 encryption before storage. Keys are encrypted at rest and never transmitted in plain text. For maximum security, you can also use hardware wallets or external signers.'
  },
  {
    question: 'Which wallets and chains are supported?',
    answer: 'We support all EVM-compatible chains (Ethereum, Polygon, Arbitrum, Optimism, Base, etc.) and Solana. You can import existing wallets or generate new ones directly in the platform.'
  },
  {
    question: 'How does transaction signing work?',
    answer: 'Transactions are prepared and displayed for review before signing. You can configure approval workflows requiring manual confirmation for high-value operations. The embedded browser also supports connecting external wallets like MetaMask.'
  },
  {
    question: 'Which platforms are integrated?',
    answer: 'We have direct API integrations with Farcaster (via Neynar) and Telegram (Bot API). For Twitter/X and other platforms, we provide browser-based automation with session isolation. Additional integrations are added regularly.'
  },
  {
    question: 'Can I self-host Web3AirdropOS?',
    answer: 'Yes! The entire platform can be self-hosted using Docker Compose. All components (backend, frontend, AI service, browser service) are containerized and can run on your own infrastructure.'
  },
  {
    question: 'What happens if an automated task fails?',
    answer: 'Failed tasks are logged with detailed error information and can be retried. The real-time terminal shows exactly what happened, and you can take manual control through the browser workspace when needed.'
  }
];

function FAQSection() {
  const [openIndex, setOpenIndex] = useState<number | null>(null);

  return (
    <section id="faq" className="py-20 px-4 bg-zinc-900/30">
      <div className="max-w-3xl mx-auto">
        {/* Section header */}
        <div className="text-center mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold text-white mb-4">
            Frequently asked questions
          </h2>
          <p className="text-lg text-zinc-400">
            Everything you need to know about Web3AirdropOS
          </p>
        </div>

        {/* FAQ list */}
        <div className="space-y-4">
          {faqs.map((faq, index) => (
            <div 
              key={index}
              className="rounded-xl bg-zinc-900/50 border border-zinc-800 overflow-hidden"
            >
              <button
                className="w-full flex items-center justify-between p-6 text-left"
                onClick={() => setOpenIndex(openIndex === index ? null : index)}
              >
                <span className="font-medium text-white">{faq.question}</span>
                <ChevronDown 
                  className={`w-5 h-5 text-zinc-400 transition-transform ${openIndex === index ? 'rotate-180' : ''}`}
                />
              </button>
              {openIndex === index && (
                <div className="px-6 pb-6">
                  <p className="text-zinc-400 leading-relaxed">{faq.answer}</p>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

// =====================================================
// FOOTER
// =====================================================
function Footer() {
  return (
    <footer className="py-12 px-4 border-t border-zinc-800">
      <div className="max-w-7xl mx-auto">
        <div className="flex flex-col md:flex-row items-center justify-between gap-6">
          {/* Logo */}
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-violet-500 to-fuchsia-500 flex items-center justify-center">
              <Zap className="w-5 h-5 text-white" />
            </div>
            <span className="text-lg font-bold text-white">Web3AirdropOS</span>
          </div>

          {/* Links */}
          <div className="flex items-center gap-8 text-sm text-zinc-400">
            <Link href="/terms" className="hover:text-white transition-colors">Terms of Service</Link>
            <Link href="/privacy" className="hover:text-white transition-colors">Privacy Policy</Link>
            <Link href="/contact" className="hover:text-white transition-colors">Contact</Link>
          </div>

          {/* Copyright */}
          <p className="text-sm text-zinc-500">
            © {new Date().getFullYear()} Web3AirdropOS. All rights reserved.
          </p>
        </div>
      </div>
    </footer>
  );
}

// =====================================================
// MAIN PAGE
// =====================================================
export default function LandingPage() {
  return (
    <div className="min-h-screen bg-zinc-950 text-white">
      <Navbar />
      <main>
        <HeroSection />
        <FeaturesSection />
        <HowItWorksSection />
        <PricingSection />
        <FAQSection />
      </main>
      <Footer />
    </div>
  );
}
