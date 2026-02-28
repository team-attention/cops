import { useMemo } from 'react'
import { timestampMs } from '@bufbuild/protobuf/wkt'
import type { FeaturedMember } from '@/gen/grpcstub/dashboard/v1/dashboard_pb'
import { formatTokenCount } from '@/shared/util/format'

interface MemberCardProps {
  member: FeaturedMember
}

// computeUptime calculates the uptime string from last_activity_at.
// Returns "Active" if within the last minute, otherwise "Xm ago", "Xh ago", etc.
function computeUptime(
  lastActivityAt: FeaturedMember['lastActivityAt'],
): string {
  if (!lastActivityAt) return '-'
  const ms = timestampMs(lastActivityAt)
  const now = Date.now()
  const diffMs = now - ms
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)

  if (diffMins < 1) return 'Active'
  if (diffMins < 60) return `${diffMins}m ago`
  return `${diffHours}h ago`
}

// isInactive returns true if last_activity_at is more than 3 minutes ago.
function isInactive(lastActivityAt: FeaturedMember['lastActivityAt']): boolean {
  if (!lastActivityAt) return true
  const ms = timestampMs(lastActivityAt)
  const now = Date.now()
  const diffMs = now - ms
  const diffMins = diffMs / 60000
  return diffMins > 3
}

export function MemberCard({ member }: MemberCardProps) {
  const uptime = useMemo(
    () => computeUptime(member.lastActivityAt),
    [member.lastActivityAt],
  )
  const inactive = useMemo(
    () => isInactive(member.lastActivityAt),
    [member.lastActivityAt],
  )

  // Total token usage: input + output + cache
  const totalTokens = useMemo(() => {
    if (!member.usage) return 0n
    return (
      member.usage.totalInputTokens +
      member.usage.totalOutputTokens +
      member.usage.totalCacheCreationTokens +
      member.usage.totalCacheReadTokens
    )
  }, [member.usage])

  // Categorize tool calls into fixed groups
  const toolCategories = useMemo(() => {
    const cats = [
      { label: 'Agent Calls', count: 0, color: 'text-orange-400' },
      { label: 'File Read', count: 0, color: 'text-cyan-400' },
      { label: 'File Write', count: 0, color: 'text-emerald-400' },
    ]
    for (const tool of member.toolCalls) {
      const name = tool.toolName.toLowerCase()
      if (name === 'agent' || name === 'sendmessage' || name === 'skill') {
        cats[0].count += tool.count
      } else if (['read', 'glob', 'grep'].includes(name)) {
        cats[1].count += tool.count
      } else if (['write', 'edit', 'notebookedit'].includes(name)) {
        cats[2].count += tool.count
      }
    }
    return cats
  }, [member.toolCalls])

  return (
    <div
      className={[
        'relative overflow-hidden rounded-xl border backdrop-blur-sm transition-all duration-300',
        inactive
          ? 'border-red-900/50 bg-red-950/10 inactive-blink'
          : 'border-l-2 border-cyan-500/40 border-t-zinc-800/50 bg-zinc-900/80 hover:border-cyan-500/60 hover:bg-zinc-900/90',
      ].join(' ')}
      style={
        inactive
          ? {
              animation: 'member-blink 1s ease-in-out infinite',
            }
          : undefined
      }
    >
      {/* Active indicator */}
      {!inactive && (
        <div className="absolute right-3 top-3 flex items-center gap-1.5">
          <div className="h-2 w-2 animate-pulse rounded-full bg-emerald-400" />
          <span className="font-mono text-[9px] uppercase tracking-wider text-zinc-500">
            Active
          </span>
        </div>
      )}

      {/* Inactive indicator */}
      {inactive && (
        <div className="absolute right-3 top-3 flex items-center gap-1.5">
          <div className="h-2 w-2 rounded-full bg-red-500" />
          <span className="font-mono text-[9px] uppercase tracking-wider text-red-400">
            Inactive
          </span>
        </div>
      )}

      <div className="p-4">
        {/* Member header */}
        <div className="mb-4">
          <h3 className="font-mono text-sm font-bold text-zinc-100 truncate pr-16">
            {member.userName || member.userId}
          </h3>
          <p className="mt-0.5 font-mono text-[10px] text-zinc-500 truncate">
            {member.projectName || member.projectId}
          </p>
        </div>

        {/* Metrics grid */}
        <div className="grid grid-cols-2 gap-3">
          {/* Prompt Count */}
          <div className="rounded-lg border border-zinc-800/80 bg-zinc-950/50 p-2.5">
            <p className="font-mono text-[9px] uppercase tracking-widest text-zinc-600">
              Prompts
            </p>
            <p className="mt-1 font-mono text-xl font-bold text-cyan-400">
              {member.promptCount.toLocaleString()}
            </p>
          </div>

          {/* Token Usage */}
          <div className="rounded-lg border border-zinc-800/80 bg-zinc-950/50 p-2.5">
            <p className="font-mono text-[9px] uppercase tracking-widest text-zinc-600">
              Tokens
            </p>
            <p className="mt-1 font-mono text-xl font-bold text-violet-400">
              {formatTokenCount(totalTokens)}
            </p>
          </div>

          {/* Uptime */}
          <div className="rounded-lg border border-zinc-800/80 bg-zinc-950/50 p-2.5">
            <p className="font-mono text-[9px] uppercase tracking-widest text-zinc-600">
              Last Active
            </p>
            <p
              className={[
                'mt-1 font-mono text-sm font-bold',
                inactive ? 'text-red-400' : 'text-emerald-400',
              ].join(' ')}
            >
              {uptime}
            </p>
          </div>

          {/* Tool Calls — categorized */}
          <div className="rounded-lg border border-zinc-800/80 bg-zinc-950/50 p-2.5">
            <p className="font-mono text-[9px] uppercase tracking-widest text-zinc-600">
              Tool Usage
            </p>
            <div className="mt-1 space-y-0.5">
              {toolCategories.map((cat) => (
                <div
                  key={cat.label}
                  className="flex items-center justify-between gap-1"
                >
                  <span
                    className={`truncate font-mono text-[9px] ${cat.color}`}
                  >
                    {cat.label}
                  </span>
                  <span className="shrink-0 font-mono text-[9px] font-bold text-amber-400">
                    {cat.count}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Bottom accent line */}
      <div
        className={[
          'absolute bottom-0 left-0 h-[2px] w-full',
          inactive
            ? 'bg-red-500/40'
            : 'bg-gradient-to-r from-cyan-500/60 to-violet-500/60',
        ].join(' ')}
      />
    </div>
  )
}
