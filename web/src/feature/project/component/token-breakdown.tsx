import { ArrowDownRight, Zap, Archive, RefreshCw } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/gen/shadcn/ui/card'
import type { TokenUsageSummary } from '@/gen/grpcstub/dashboard/v1/dashboard_pb'

interface TokenBreakdownProps {
  usage?: TokenUsageSummary
}

interface TokenCategoryProps {
  label: string
  value: bigint | undefined
  total: bigint
  color: string
  icon: React.ReactNode
  description: string
}

const formatTokenCount = (value: bigint | undefined): string => {
  if (!value) return '0'
  const num = Number(value)
  if (num >= 1_000_000) return `${(num / 1_000_000).toFixed(2)}M`
  if (num >= 1_000) return `${(num / 1_000).toFixed(1)}K`
  return num.toLocaleString()
}

const TokenCategory = ({ label, value, total, color, icon, description }: TokenCategoryProps) => {
  const numValue = Number(value ?? 0n)
  const numTotal = Number(total)
  const percentage = numTotal > 0 ? (numValue / numTotal) * 100 : 0

  return (
    <div className="group relative space-y-3 rounded-lg border border-zinc-800/50 bg-zinc-950/50 p-4 transition-all duration-300 hover:border-zinc-700/50 hover:bg-zinc-900/50">
      {/* Glow on hover */}
      <div
        className="pointer-events-none absolute inset-0 rounded-lg opacity-0 blur-xl transition-opacity duration-500 group-hover:opacity-10"
        style={{ background: `radial-gradient(circle at 50% 50%, ${color}, transparent 70%)` }}
      />

      <div className="relative flex items-center justify-between">
        <div className="flex items-center gap-2">
          <div
            className="rounded-md p-1.5 transition-colors duration-300"
            style={{ backgroundColor: `${color}15` }}
          >
            <div style={{ color }}>{icon}</div>
          </div>
          <div>
            <p className="text-xs font-medium uppercase tracking-wider text-zinc-400">{label}</p>
            <p className="font-mono text-xs text-zinc-600">{description}</p>
          </div>
        </div>
        <div className="text-right">
          <p className="font-mono text-xl font-bold" style={{ color }}>
            {formatTokenCount(value)}
          </p>
          <p className="font-mono text-[10px] text-zinc-600">
            {percentage.toFixed(1)}%
          </p>
        </div>
      </div>

      {/* Progress bar */}
      <div className="relative h-2 overflow-hidden rounded-full bg-zinc-800">
        <div
          className="h-full rounded-full transition-all duration-700 ease-out"
          style={{
            width: `${percentage}%`,
            background: `linear-gradient(90deg, ${color}80, ${color})`,
            boxShadow: `0 0 12px ${color}40`,
          }}
        />
        {/* Animated shimmer */}
        <div
          className="absolute inset-0 -translate-x-full animate-[shimmer_2s_infinite] bg-gradient-to-r from-transparent via-white/10 to-transparent"
          style={{ animationDelay: `${Math.random() * 2}s` }}
        />
      </div>
    </div>
  )
}

export const TokenBreakdown = ({ usage }: TokenBreakdownProps) => {
  const inputTokens = usage?.totalInputTokens ?? 0n
  const outputTokens = usage?.totalOutputTokens ?? 0n
  const cacheCreation = usage?.totalCacheCreationTokens ?? 0n
  const cacheRead = usage?.totalCacheReadTokens ?? 0n
  const total = inputTokens + outputTokens + cacheCreation + cacheRead

  const categories: Omit<TokenCategoryProps, 'total'>[] = [
    {
      label: 'Input Tokens',
      value: inputTokens,
      color: '#22d3ee',
      icon: <ArrowDownRight className="h-4 w-4" />,
      description: 'Tokens sent to model',
    },
    {
      label: 'Output Tokens',
      value: outputTokens,
      color: '#a78bfa',
      icon: <Zap className="h-4 w-4" />,
      description: 'Tokens generated',
    },
    {
      label: 'Cache Creation',
      value: cacheCreation,
      color: '#fbbf24',
      icon: <Archive className="h-4 w-4" />,
      description: 'Cache write tokens',
    },
    {
      label: 'Cache Read',
      value: cacheRead,
      color: '#34d399',
      icon: <RefreshCw className="h-4 w-4" />,
      description: 'Cache hit tokens',
    },
  ]

  return (
    <Card className="group relative overflow-hidden border-zinc-800/50 bg-zinc-900/80 backdrop-blur-sm transition-all duration-300 hover:border-zinc-700/50">
      {/* Ambient corner glow */}
      <div className="pointer-events-none absolute -right-16 -top-16 h-32 w-32 rounded-full bg-cyan-500/5 blur-3xl transition-opacity duration-500 group-hover:bg-cyan-500/10" />

      <CardHeader className="border-b border-zinc-800/50 pb-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="rounded-lg border border-cyan-500/20 bg-cyan-500/10 p-2">
              <Zap className="h-4 w-4 text-cyan-400" />
            </div>
            <div>
              <CardTitle className="text-sm font-semibold text-zinc-100">
                Token Breakdown
              </CardTitle>
              <p className="font-mono text-[10px] text-zinc-600">
                Usage distribution by category
              </p>
            </div>
          </div>
          <div className="text-right">
            <p className="font-mono text-xs text-zinc-500">Total</p>
            <p className="font-mono text-lg font-bold text-zinc-100">
              {formatTokenCount(total)}
            </p>
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-3 pt-4">
        {categories.map((category) => (
          <TokenCategory key={category.label} {...category} total={total} />
        ))}

        {/* Visual summary bar */}
        <div className="mt-4 space-y-2">
          <p className="font-mono text-[10px] uppercase tracking-wider text-zinc-600">
            Distribution
          </p>
          <div className="flex h-3 overflow-hidden rounded-full bg-zinc-800">
            {categories.map((category, index) => {
              const percentage = Number(total) > 0
                ? (Number(category.value ?? 0n) / Number(total)) * 100
                : 0
              return (
                <div
                  key={category.label}
                  className="relative h-full transition-all duration-700"
                  style={{
                    width: `${percentage}%`,
                    backgroundColor: category.color,
                    marginLeft: index > 0 ? '1px' : 0,
                  }}
                  title={`${category.label}: ${percentage.toFixed(1)}%`}
                />
              )
            })}
          </div>
          <div className="flex flex-wrap justify-center gap-4 pt-1">
            {categories.map((category) => (
              <div key={category.label} className="flex items-center gap-1.5">
                <div
                  className="h-2 w-2 rounded-full"
                  style={{ backgroundColor: category.color }}
                />
                <span className="font-mono text-[10px] text-zinc-500">
                  {category.label}
                </span>
              </div>
            ))}
          </div>
        </div>
      </CardContent>

      {/* Bottom accent */}
      <div className="absolute bottom-0 left-0 h-[2px] w-0 bg-gradient-to-r from-cyan-500 via-violet-500 to-emerald-500 transition-all duration-700 group-hover:w-full" />
    </Card>
  )
}
