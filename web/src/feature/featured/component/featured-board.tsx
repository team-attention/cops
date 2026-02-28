import { MemberCard } from './member-card'
import { RaceTrack } from './race-track'
import type { FeaturedMember } from '@/gen/grpcstub/dashboard/v1/dashboard_pb'

interface FeaturedBoardProps {
  members: Array<FeaturedMember>
  sinceUnix: bigint
}

// FeaturedBoard renders a race track visualization and member cards.
export function FeaturedBoard({ members, sinceUnix }: FeaturedBoardProps) {
  return (
    <div className="space-y-8">
      {/* Race Track */}
      <RaceTrack members={members} sinceUnix={sinceUnix} />

      {/* Member Cards Grid */}
      {members.length > 0 && (
        <div>
          <div className="mb-3 flex items-center gap-2">
            <span className="font-mono text-[10px] uppercase tracking-widest text-zinc-600">
              // Member Stats
            </span>
          </div>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {members.map((member) => (
              <MemberCard key={member.userId} member={member} />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
