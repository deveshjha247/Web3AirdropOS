'use client'

import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import {
  Sparkles,
  Send,
  Copy,
  RefreshCw,
  Clock,
  FileText,
  Calendar,
  TrendingUp,
} from 'lucide-react'

const platforms = ['farcaster', 'twitter', 'telegram', 'discord']
const tones = ['professional', 'casual', 'witty', 'informative', 'engaging']
const types = ['post', 'reply', 'thread']

export default function ContentPage() {
  const [platform, setPlatform] = useState('farcaster')
  const [tone, setTone] = useState('casual')
  const [type, setType] = useState('post')
  const [prompt, setPrompt] = useState('')
  const [isGenerating, setIsGenerating] = useState(false)
  const [generatedContent, setGeneratedContent] = useState<string[]>([])

  const handleGenerate = async () => {
    setIsGenerating(true)
    // Simulate API call
    await new Promise((resolve) => setTimeout(resolve, 2000))
    setGeneratedContent([
      "GM frens! 🌅 Just shipped a new feature that's going to change how we think about on-chain governance. Thread incoming 🧵",
      "Been building in the trenches for 6 months. Finally ready to share what we've been cooking. The future of Web3 is community-owned infrastructure. Here's why 👇",
      "Hot take: The next bull run will be driven by actual utility, not speculation. Protocols solving real problems are going to win big. What are you building? 🔨",
    ])
    setIsGenerating(false)
  }

  const drafts = [
    {
      id: '1',
      content: 'Just deployed our new smart contract on Base! Gas fees are insanely low 🔵',
      platform: 'farcaster',
      scheduledFor: null,
      createdAt: '2 hours ago',
    },
    {
      id: '2',
      content: 'Thread: 10 lessons learned from building in Web3 for 2 years 🧵',
      platform: 'twitter',
      scheduledFor: '2024-01-20 14:00',
      createdAt: '1 day ago',
    },
  ]

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">AI Content Studio</h1>
          <p className="text-muted-foreground">
            Generate platform-optimized content with AI
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline">
            <Calendar className="h-4 w-4 mr-2" />
            Scheduled
          </Button>
          <Button variant="outline">
            <FileText className="h-4 w-4 mr-2" />
            Drafts
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Generator */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Sparkles className="h-5 w-5 text-primary" />
              Content Generator
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Platform Selection */}
            <div>
              <label className="text-sm font-medium mb-2 block">Platform</label>
              <div className="flex gap-2 flex-wrap">
                {platforms.map((p) => (
                  <Button
                    key={p}
                    variant={platform === p ? 'default' : 'outline'}
                    size="sm"
                    onClick={() => setPlatform(p)}
                    className="capitalize"
                  >
                    {p}
                  </Button>
                ))}
              </div>
            </div>

            {/* Type Selection */}
            <div>
              <label className="text-sm font-medium mb-2 block">Type</label>
              <div className="flex gap-2">
                {types.map((t) => (
                  <Button
                    key={t}
                    variant={type === t ? 'default' : 'outline'}
                    size="sm"
                    onClick={() => setType(t)}
                    className="capitalize"
                  >
                    {t}
                  </Button>
                ))}
              </div>
            </div>

            {/* Tone Selection */}
            <div>
              <label className="text-sm font-medium mb-2 block">Tone</label>
              <div className="flex gap-2 flex-wrap">
                {tones.map((t) => (
                  <Button
                    key={t}
                    variant={tone === t ? 'default' : 'outline'}
                    size="sm"
                    onClick={() => setTone(t)}
                    className="capitalize"
                  >
                    {t}
                  </Button>
                ))}
              </div>
            </div>

            {/* Prompt */}
            <div>
              <label className="text-sm font-medium mb-2 block">
                What would you like to post about?
              </label>
              <textarea
                value={prompt}
                onChange={(e) => setPrompt(e.target.value)}
                placeholder="e.g., New DeFi protocol launch, thoughts on L2 scaling, community update..."
                className="w-full h-32 p-3 rounded-lg bg-muted border border-input text-sm focus:outline-none focus:ring-2 focus:ring-primary resize-none"
              />
            </div>

            {/* Generate Button */}
            <Button
              className="w-full"
              onClick={handleGenerate}
              disabled={isGenerating}
            >
              {isGenerating ? (
                <>
                  <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                  Generating...
                </>
              ) : (
                <>
                  <Sparkles className="h-4 w-4 mr-2" />
                  Generate Content
                </>
              )}
            </Button>
          </CardContent>
        </Card>

        {/* Generated Content */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              <span>Generated Options</span>
              {generatedContent.length > 0 && (
                <Button variant="ghost" size="sm" onClick={handleGenerate}>
                  <RefreshCw className="h-4 w-4 mr-1" />
                  Regenerate
                </Button>
              )}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {generatedContent.length > 0 ? (
              <div className="space-y-4">
                {generatedContent.map((content, index) => (
                  <div
                    key={index}
                    className="p-4 rounded-lg bg-muted border border-border"
                  >
                    <p className="text-sm mb-3">{content}</p>
                    <div className="flex items-center justify-between">
                      <span className="text-xs text-muted-foreground">
                        {content.length} characters
                      </span>
                      <div className="flex gap-2">
                        <Button variant="ghost" size="sm">
                          <Copy className="h-3 w-3 mr-1" />
                          Copy
                        </Button>
                        <Button variant="ghost" size="sm">
                          <Clock className="h-3 w-3 mr-1" />
                          Schedule
                        </Button>
                        <Button size="sm">
                          <Send className="h-3 w-3 mr-1" />
                          Post
                        </Button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center h-64 text-muted-foreground">
                <Sparkles className="h-12 w-12 mb-4" />
                <p>Generated content will appear here</p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Drafts */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileText className="h-5 w-5" />
            Recent Drafts
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {drafts.map((draft) => (
              <div
                key={draft.id}
                className="flex items-start justify-between p-4 rounded-lg bg-muted"
              >
                <div className="flex-1">
                  <p className="text-sm">{draft.content}</p>
                  <div className="flex items-center gap-4 mt-2 text-xs text-muted-foreground">
                    <span className="capitalize">{draft.platform}</span>
                    <span>{draft.createdAt}</span>
                    {draft.scheduledFor && (
                      <span className="flex items-center gap-1 text-primary">
                        <Clock className="h-3 w-3" />
                        {draft.scheduledFor}
                      </span>
                    )}
                  </div>
                </div>
                <div className="flex gap-2">
                  <Button variant="ghost" size="sm">
                    Edit
                  </Button>
                  <Button size="sm">
                    <Send className="h-3 w-3 mr-1" />
                    Post
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
