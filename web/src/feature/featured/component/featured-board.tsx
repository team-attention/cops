import { MemberCard } from './member-card'
import type { FeaturedMember } from '@/gen/grpcstub/dashboard/v1/dashboard_pb'

interface FeaturedBoardProps {
  members: Array<FeaturedMember>
}

// FeaturedBoard renders a grid of MemberCard components for each featured member.
export function FeaturedBoard({ members }: FeaturedBoardProps) {
  if (members.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-24 text-zinc-500">
        <p className="font-mono text-sm">No active members found</p>
        <p className="mt-2 font-mono text-xs text-zinc-600">
          Members will appear here once they start Claude Code sessions
        </p>
      </div>
    )
  }

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      {members.map((member) => (
        <MemberCard key={member.userId} member={member} />
      ))}
    </div>
  )
}
