import { ArrowDownRight, ArrowUpRight, MessageSquare, Zap, Archive } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/gen/shadcn/ui/card'
import type { TokenUsageSummary } from '@/gen/grpcstub/dashboard/v1/dashboard_pb'

interface OverviewStatsProps {
  totalUsage?: TokenUsageSummary
  projectCount: number
  sessionCount: number
}

const formatTokenCount = (value: bigint | undefined): string => {
  if (!value) return '0'
  const num = Number(value)
  if (num >= 1_000_000) return `${(num / 1_000_000).toFixed(2)}M`
  if (num >= 1_000) return `${(num / 1_000).toFixed(1)}K`
  return num.toLocaleString()
}

interface StatCardProps {
  title: string
  value: string
  subtitle?: string
  icon: React.ReactNode
  trend?: 'up' | 'down' | 'neutral'
  accentColor: string
}

const StatCard = ({ title, value, subtitle, icon, trend, accentColor }: StatCardProps) => (
  <Card className="group relative overflow-hidden border-zinc-800/50 bg-zinc-900/80 backdrop-blur-sm transition-all duration-300 hover:border-zinc-700/80 hover:bg-zinc-900/90">
    {/* Glow effect on hover */}
    <div
      className="absolute inset-0 opacity-0 blur-xl transition-opacity duration-500 group-hover:opacity-20"
      style={{ background: `radial-gradient(circle at 50% 50%, ${accentColor}, transparent 70%)` }}
    />

    {/* Scanline effect */}
    <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(transparent_50%,rgba(0,0,0,0.1)_50%)] bg-[length:100%_4px] opacity-0 transition-opacity duration-300 group-hover:opacity-30" />

    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
      <CardTitle className="text-xs font-medium uppercase tracking-widest text-zinc-500">
        {title}
      </CardTitle>
      <div
        className="rounded-md p-2 transition-colors duration-300"
        style={{ backgroundColor: `${accentColor}15` }}
      >
        <div style={{ color: accentColor }}>{icon}</div>
      </div>
    </CardHeader>
    <CardContent>
      <div className="flex items-baseline gap-2">
        <span
          className="font-mono text-3xl font-bold tracking-tight transition-colors duration-300"
          style={{ color: accentColor }}
        >
          {value}
        </span>
        {trend && trend !== 'neutral' && (
          <span className={`flex items-center text-xs ${trend === 'up' ? 'text-emerald-400' : 'text-rose-400'}`}>
            {trend === 'up' ? <ArrowUpRight className="h-3 w-3" /> : <ArrowDownRight className="h-3 w-3" />}
          </span>
        )}
      </div>
      {subtitle && (
        <p className="mt-1 font-mono text-xs text-zinc-600">{subtitle}</p>
      )}
    </CardContent>

    {/* Bottom accent line */}
    <div
      className="absolute bottom-0 left-0 h-[2px] w-0 transition-all duration-500 group-hover:w-full"
      style={{ backgroundColor: accentColor }}
    />
  </Card>
)

export const OverviewStats = ({ totalUsage, projectCount, sessionCount }: OverviewStatsProps) => {
  const inputTokens = formatTokenCount(totalUsage?.totalInputTokens)
  const outputTokens = formatTokenCount(totalUsage?.totalOutputTokens)
  const cacheTokens = formatTokenCount(
    totalUsage
      ? (totalUsage.totalCacheCreationTokens ?? 0n) + (totalUsage.totalCacheReadTokens ?? 0n)
      : undefined
  )

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      <StatCard
        title="Input Tokens"
        value={inputTokens}
        subtitle="tokens consumed"
        icon={<ArrowDownRight className="h-4 w-4" />}
        accentColor="#22d3ee"
      />
      <StatCard
        title="Output Tokens"
        value={outputTokens}
        subtitle="tokens generated"
        icon={<Zap className="h-4 w-4" />}
        accentColor="#a78bfa"
      />
      <StatCard
        title="Cache Tokens"
        value={cacheTokens}
        subtitle="cached I/O"
        icon={<Archive className="h-4 w-4" />}
        accentColor="#fbbf24"
      />
      <StatCard
        title="Total Sessions"
        value={sessionCount.toLocaleString()}
        subtitle={`across ${projectCount} projects`}
        icon={<MessageSquare className="h-4 w-4" />}
        accentColor="#34d399"
      />
    </div>
  )
}
